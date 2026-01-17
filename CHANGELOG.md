# Changelog
Все значимые изменения проекта WipeDisk Enterprise фиксируются в этом файле.

Формат основан на: https://keepachangelog.com/ru/1.1.0/  
Версионирование: SemVer (MAJOR.MINOR.PATCH)

---

## [1.2.1.1] - 2026-01-17

### Fixed
- **Architecture compliance**: Removed Linux syscalls from internal/system/disk.go, ensuring Windows-only compatibility
- **Struct field alignment**: Fixed WipeOperation field references from `op.Target` to `op.Disk` across reporting modules
- **Function naming conflicts**: Resolved duplicate `FillPattern` function by renaming to `FillBufferPattern` in internal/wipe/buffer_pool.go
- **Import dependencies**: Added missing internal/cli and internal/system imports in enterprise reporting module
- **Build errors**: Eliminated undefined `config.EnsureDirectories` function call in main.go
- **Parameter mismatches**: Fixed generateAndSaveReport function signature inconsistencies

### Changed
- **Enterprise reporting**: Enhanced security audit reports with Russian language localization
- **Risk assessment**: Updated risk levels to Russian (Низкий/Средний/Высокий/Критический)
- **Category names**: Translated security categories to Russian (Остатки данных, Системные артефакты, etc.)
- **Report metadata**: Updated report titles and descriptions for Russian enterprise environments
- **Cleanup integration**: Added system cleanup operations to enterprise reporting framework

### Added
- **System cleanup operations**: Implemented comprehensive cleanup module with print queue, DNS cache, browser cache, temp files, and old logs cleanup
- **CLI cleanup commands**: Added `wipedisk cleanup` command with operation listing, category-based execution, and dry-run support
- **Enterprise cleanup categories**: New "Системная очистка" category in security audit reports
- **Browser cleanup support**: Automatic cleanup for Chrome, Firefox, and Yandex Browser caches and cookies
- **Print queue management**: Safe print queue cleanup with service restart functionality
- **DNS cache management**: Complete DNS cache reset with winsock repair
- **Maintenance integration**: Cleanup operations integrated with existing maintenance framework

### Security
- **Enhanced data remnant detection**: Improved identification of incomplete wipe operations
- **System artifact analysis**: Better detection of system configuration exposure risks
- **Temporary file monitoring**: Enhanced tracking of temp file creation and cleanup
- **Browser privacy protection**: Comprehensive browser data cleanup for enterprise environments
- **Print queue security**: Secure cleanup of potentially sensitive print job remnants

### Stability
- **Error handling**: Improved graceful error handling in cleanup operations
- **Service management**: Robust print queue service restart with proper error recovery
- **Resource cleanup**: Enhanced memory and file handle management in cleanup operations
- **Concurrent operations**: Thread-safe cleanup operation execution
- **Logging integration**: Enterprise-grade logging for all cleanup activities

### Performance
- **Optimized cleanup sequencing**: Efficient cleanup operation ordering for minimal system impact
- **Parallel cleanup support**: Foundation for concurrent cleanup operations
- **Resource monitoring**: Real-time resource usage tracking during cleanup operations
- **Cache optimization**: Improved browser cache cleanup performance
- **Memory efficiency**: Reduced memory footprint in cleanup operations

---

## [1.2.1] — 14.01.2026
### Added
- **Безопасное затирание системного диска** через флаг `--allow-system-disk`
  - Политика безопасности для системного диска (только %TEMP%, %WINDIR%\Temp)
  - Автоматическое определение SSD и рекомендация cipher /w
  - Ограничения: 2GB temp files, 30 минут timeout
- **Verify режим** — проверка качества затирания
  - Уровни проверки: basic, physical, aggressive
  - Физическая верификация с многократными попытками чтения
  - Анализ аномалий и соответствие стандартам (DOD5220, NIST800-88, BSI_VSITR)
  - Отчёты в JSON и CSV форматах
- **Maintenance режим** — единый режим обслуживания
  - Предопределенные планы: full_year, light_monthly, security_quarterly, quick_cleanup, deep_clean, verify_only
  - Параллельное и последовательное выполнение фаз
  - Orchestrator с graceful shutdown и таймаутами
  - Фазы: clean_temp, clean_update_cache, clean_browsers, wipe_free_space, optimize_disk, verify_wipe
- **Self-diagnose режим** — системная диагностика
  - Тесты: permissions, disks, memory, cpu, paths, api, wipe, network
  - Уровни: quick, full, deep
  - Детальные отчёты о состоянии системы
  - Предсказание проблем и рекомендации
- **Professional bat-меню** с динамической генерацией
  - Автоопределение возможностей системы (SSD, admin rights, cipher)
  - Smart preview команд перед выполнением
  - Поддержка GPO развертывания
  - Логирование всех действий
- **Улучшенный throttling** с адаптивными алгоритмами
  - Корректная работа на старых HDD
  - Защита от перегрузки системы
  - Динамическая адаптация скорости

### Changed
- **Архитектура** — полный рефакторинг в модульную структуру
  - Удалены дублирующие реализации
  - Чистое разделение ответственности
  - Унифицированные интерфейсы
- **CLI** — переработаны все команды
  - Поддержка `--profile`, `--engine`, `--silent`, `--max-duration`
  - Улучшенная обработка ошибок
  - Graceful shutdown по Ctrl+C
- **Безопасность** — усилены защиты для доменных сред
  - Проверка прав администратора
  - Безопасные пути по умолчанию
  - Защита от случайного затирания системных файлов
- **Стабильность** — исправлены все критические ошибки
  - Корректная работа контекстов и отмены
  - Предотвращение deadlock и race conditions
  - Улучшенная обработка ошибок Windows API

### Fixed
- **Критические ошибки компиляции** — все ошибки return statements исправлены
- **Memory leaks** — исправлены утечки памяти в долгих операциях
- **Throttling** — исправлены некорректные задержки на высоких скоростях
- **SSD оптимизация** — корректная работа с TRIM и оптимизацией
- **Logging** — исправлены проблемы с кодировкой UTF-8 в логах
- **JSON отчёты** — исправлена сериализация сложных структур
- **GPO совместимость** — исправлены проблемы с системными путями

### Security
- Добавлены проверки безопасности для системного диска
- Усилены проверки прав доступа
- Защита от выполнения на серверных ОС без разрешения
- Валидация всех пользовательских входов

---

## [1.2.0] — 13-01-2026
### Added
- Полная поддержка всех CLI-режимов через bat-меню
- Silent-режим запуска (`--silent`, без подтверждений и интерактива)
- Поддержка всех engines:
  - internal
  - sdelete-compatible
  - cipher (/w через Windows)
- Поддержка профилей:
  - safe
  - balanced
  - aggressive
  - sdelete
- Расширенное bat-меню для всех сценариев запуска
- Проектный CHANGELOG.md

### Changed
- Обновлён README.md под v1.2.0
- Унифицированы команды запуска во всех bat-файлах
- Улучшена структура проекта
- Удалены устаревшие параметры и режимы

### Fixed
- Устранены расхождения между CLI и bat-меню
- Исправлены некорректные команды запуска в bat
- Исправлены проблемы с тихим режимом

---

## [1.1.1] — 12.01.2026

### 🔧 Fixed
- Исправлена обработка `context.Context` — корректные статусы PARTIAL / CANCELLED / FAILED  
- Исправлены ошибки lifecycle wipe — паузы и таймауты больше не приводят к FAILED  
- Добавлена строгая валидация конфигурации (все поля проверяются при запуске)  
- Устранены возможные panic:  
  - rand.Int63n(0)  
  - nil-указатели  
  - деление на 0  
- Исправлена логика прерывания:  
  - Ctrl+C → CANCELLED  
  - max-duration → PARTIAL  

---

### 🛡️ Security & Stability
- Строгая модель статусов: COMPLETED / PARTIAL / CANCELLED / FAILED  
- Исправлен throttling записи (устранены зависания и 0 KB/s)  
- Гарантирована реальная запись: `rand.Read → Write (partial aware) → Sync → Close`  
- Защита от edge-case размеров файлов  
- Валидация методов wipe: random / zero / dod5220 / sdelete-compatible  

---

### 🔄 Architecture
- Полностью переработанная wipe-логика (безопасное создание множества tmp-файлов)  
- Поддержка движков:  
  - internal  
  - sdelete-compatible  
  - cipher (Windows `cipher /w`)  
- Добавлены профили производительности:  
  - safe  
  - balanced  
  - aggressive  
  - sdelete  
- Реализована JSON-отчётность + агрегация  
- Улучшен EnterpriseLogger (fallback в stdout при проблемах с правами)  

---

### 🚀 Added
- Новый engine: `--engine cipher`  
- Новый режим: `--engine sdelete-compatible`  
- Поддержка `--profile safe|balanced|aggressive|sdelete`  
- Адаптивный chunk size для HDD / SSD  
- Буферизированная запись с корректным throttling  

---

### 📦 BAT updates
- Переименование бинарника:  
  - `wipedisk_enterprise.exe` → `wipedisk.exe`  
- Корректная обработка exit codes:  
  - 0 — success  
  - 1 — error  
  - 2 — warnings  
- Обновлены все bat-скрипты (основной, сетевой, планировщик, silent)  
- Улучшено логирование в bat  

---

### 🗑️ Removed
- Удалён дублирующий файл `wipe_new.go`  
- Удалена старая сборка `wipedisk_enterprise.exe`  
- Очищены устаревшие лог-файлы  


## [1.0.0] — 30-12.2025
### Added
- Первая рабочая версия утилиты
- Очистка temp/caches
- Затирание свободного места через tmp-файлы
- Поддержка dry-run
- Логирование в файл
- CLI на Cobra

---

## Принципы ведения changelog
- Каждая версия фиксируется перед релизом
- Все изменения группируются:
  - Added — новое
  - Changed — изменения
  - Fixed — исправления
  - Removed — удалённое
- Формат версий: X.Y.Z (SemVer)

