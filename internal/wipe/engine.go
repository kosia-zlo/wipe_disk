package wipe

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"wipedisk_enterprise/internal/config"
	"wipedisk_enterprise/internal/logging"
	"wipedisk_enterprise/internal/system"
)

type WipeEngine string

const (
	EngineInternal      WipeEngine = "internal"
	EngineSDeleteCompat WipeEngine = "sdelete-compatible"
	EngineCipher        WipeEngine = "cipher"
)

// WipeWithEngine выполняет затирание с указанным движком и режимом
func WipeWithEngine(ctx context.Context, disk system.DiskInfo, cfg *config.Config, logger *logging.EnterpriseLogger, dryRun bool, maxDuration time.Duration, engine WipeEngine, mode WipeMode, profile string) *WipeOperation {
	switch engine {
	case EngineCipher:
		// cipher engine всегда использует cipher mode
		return wipeWithCipher(ctx, disk, logger, dryRun, maxDuration)
	case EngineSDeleteCompat:
		// sdelete-compatible использует sdelete mode
		return wipeWithStrategy(ctx, disk, cfg, logger, dryRun, maxDuration, ModeSDelete, profile)
	case EngineInternal:
		fallthrough
	default:
		// internal engine использует указанный mode
		return wipeWithStrategy(ctx, disk, cfg, logger, dryRun, maxDuration, mode, profile)
	}
}

// wipeWithCipher использует Windows cipher /w для затирания свободного места
func wipeWithCipher(ctx context.Context, disk system.DiskInfo, logger *logging.EnterpriseLogger, dryRun bool, maxDuration time.Duration) *WipeOperation {
	op := &WipeOperation{
		ID:        fmt.Sprintf("cipher_%d", time.Now().UnixNano()),
		Disk:      disk.Letter,
		Method:    "cipher",
		Passes:    1,
		ChunkSize: 0, // Не применимо для cipher
		Status:    "RUNNING",
		StartTime: time.Now(),
	}

	logger.Log("INFO", "Запуск Windows Cipher для затирания", "disk", disk.Letter, "engine", "cipher")

	if dryRun {
		op.Status = "COMPLETED"
		op.BytesWiped = disk.FreeSize
		now := time.Now()
		op.EndTime = &now
		op.SpeedMBps = 50.0 // Примерная скорость для cipher
		logger.Log("INFO", "DRY RUN: затирание cipher завершено", "disk", disk.Letter, "bytes", op.BytesWiped)
		return op
	}

	// Проверяем доступность cipher
	if _, err := exec.LookPath("cipher.exe"); err != nil {
		op.Status = "FAILED"
		op.Error = fmt.Sprintf("cipher.exe не найден: %v", err)
		logger.Log("ERROR", "cipher.exe недоступен", "disk", disk.Letter, "error", err.Error())
		now := time.Now()
		op.EndTime = &now
		return op
	}

	// Запускаем cipher /w:D:
	cmd := exec.CommandContext(ctx, "cipher.exe", "/w:"+strings.TrimSuffix(disk.Letter, "\\"))

	// Перехватываем stdout/stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		op.Status = "FAILED"
		op.Error = fmt.Sprintf("ошибка создания stdout pipe: %v", err)
		logger.Log("ERROR", "ошибка запуска cipher", "disk", disk.Letter, "error", err.Error())
		now := time.Now()
		op.EndTime = &now
		return op
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		op.Status = "FAILED"
		op.Error = fmt.Sprintf("ошибка создания stderr pipe: %v", err)
		logger.Log("ERROR", "ошибка запуска cipher", "disk", disk.Letter, "error", err.Error())
		now := time.Now()
		op.EndTime = &now
		return op
	}

	// Запускаем команду
	if err := cmd.Start(); err != nil {
		op.Status = "FAILED"
		op.Error = fmt.Sprintf("ошибка запуска cipher: %v", err)
		logger.Log("ERROR", "ошибка запуска cipher", "disk", disk.Letter, "error", err.Error())
		now := time.Now()
		op.EndTime = &now
		return op
	}

	// Читаем вывод в отдельных горутинах
	done := make(chan bool, 2)

	// Читаем stdout
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			logger.Log("INFO", "cipher stdout", "disk", disk.Letter, "output", line)
			// Парсим прогресс если возможно
			if strings.Contains(line, "Writing") || strings.Contains(line, "Wiping") {
				fmt.Printf("\r[Cipher %s] %s", disk.Letter, line)
			}
		}
		done <- true
	}()

	// Читаем stderr
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			logger.Log("WARN", "cipher stderr", "disk", disk.Letter, "output", line)
		}
		done <- true
	}()

	// Ожидаем завершения или отмены
	cmdDone := make(chan error, 1)
	go func() {
		cmdDone <- cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		// Контекст отменен
		if ctx.Err() == context.DeadlineExceeded {
			op.Status = "PARTIAL"
			op.Warning = "Операция прервана по таймауту"
			logger.Log("WARN", "cipher прерван по таймауту", "disk", disk.Letter)
		} else {
			op.Status = "CANCELLED"
			op.Warning = "Операция отменена пользователем"
			logger.Log("WARN", "cipher отменен пользователем", "disk", disk.Letter)
		}

		// Пытаемся корректно завершить процесс
		if cmd.Process != nil {
			cmd.Process.Kill()
		}

	case err := <-cmdDone:
		// Команда завершилась
		now := time.Now()
		op.EndTime = &now

		if err != nil {
			if ctx.Err() != nil {
				// Если контекст уже отменен, это не ошибка
				if ctx.Err() == context.DeadlineExceeded {
					op.Status = "PARTIAL"
					op.Warning = "Операция прервана по таймауту"
				} else {
					op.Status = "CANCELLED"
					op.Warning = "Операция отменена пользователем"
				}
			} else {
				op.Status = "FAILED"
				op.Error = fmt.Sprintf("cipher завершился с ошибкой: %v", err)
				logger.Log("ERROR", "cipher завершился с ошибкой", "disk", disk.Letter, "error", err.Error())
			}
		} else {
			op.Status = "COMPLETED"
			op.BytesWiped = disk.FreeSize // cipher затирает все свободное место
			if op.EndTime.Sub(op.StartTime).Seconds() > 0 {
				op.SpeedMBps = float64(op.BytesWiped) / (1024 * 1024) / op.EndTime.Sub(op.StartTime).Seconds()
			}
			logger.Log("INFO", "cipher успешно завершился", "disk", disk.Letter, "bytes", op.BytesWiped, "speed", op.SpeedMBps)
		}
	}

	// Ожидаем завершения чтения вывода
	<-done
	<-done

	fmt.Printf("\n[Cipher %s] %s\n", disk.Letter, op.Status)
	return op
}
