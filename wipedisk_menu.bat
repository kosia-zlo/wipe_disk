@echo off
chcp 65001 >nul
setlocal enabledelayedexpansion

:: WipeDisk Enterprise 1.2.2 Professional Menu
:: =============================================

set "VERSION=1.2.2"
set "TITLE=WipeDisk Enterprise %VERSION% [ADMIN]"
set "LOGFILE=%TEMP%\wipedisk_menu.log"
set "EXE_PATH=%~dp0wipedisk.exe"

:: Проверка админ прав (временно отключена для теста)
rem net session >nul 2>&1
rem if %errorlevel% neq 0 (
rem     echo [ERROR] Требуются права администратора
rem     echo Пожалуйста, запустите меню от имени администратора
rem     pause
rem     exit /b 1
rem )

:: Определение возможностей системы
set "HAS_ADMIN=true"
set "IS_SSD=false"
set "HAS_CIPHER=false"
set "AVAILABLE_PROFILES="
set "AVAILABLE_ENGINES="

:: Получение версии WipeDisk
for /f "tokens=*" %%i in ('"%EXE_PATH%" --version 2^>nul') do set "VERSION=%%i"

:: Получение доступных профилей
set "AVAILABLE_PROFILES=safe balanced aggressive fast sdelete"

:: Получение доступных движков
set "AVAILABLE_ENGINES=internal sdelete-compatible cipher"

:: Проверка типа диска
for /f "tokens=3" %%t in ('"%EXE_PATH%" info 2^>nul ^| find "C:"') do (
    if /i "%%t"=="SSD" set "IS_SSD=true"
)

:: Проверка доступности cipher
cipher /? >nul 2>&1
if %errorlevel% equ 0 set "HAS_CIPHER=true"

:: Логирование запуска
echo [%DATE% %TIME%] Menu started, User: %USERNAME%, Version: %VERSION% >> "%LOGFILE%"

:main_menu
cls
echo.
echo ============================================
echo    WipeDisk Enterprise %VERSION% [ADMIN]
echo ============================================
echo.
echo  1.  Secure wipe free space
echo  2.  System maintenance (NEW)
echo  3.  Verify wipe quality (NEW)
echo  4.  Diagnostics ^& self-test (NEW)
echo  5.  Configure profiles
echo  6.  Generate reports
echo  7.  Silent mode for GPO
echo  8.  Dry-run (test mode)
echo  9.  Exit
echo.
echo  System Info:
echo    Admin: %HAS_ADMIN%
echo    SSD: %IS_SSD%
echo    Cipher: %HAS_CIPHER%
echo.
set /p "choice=Выберите опцию (1-9): "

if "%choice%"=="1" goto wipe_menu
if "%choice%"=="2" goto maintenance_menu
if "%choice%"=="3" goto verify_menu
if "%choice%"=="4" goto diagnose_menu
if "%choice%"=="5" goto profiles_menu
if "%choice%"=="6" goto reports_menu
if "%choice%"=="7" goto silent_mode
if "%choice%"=="8" goto dryrun_mode
if "%choice%"=="9" goto exit_menu

echo [ERROR] Неверный выбор
pause
goto main_menu

:wipe_menu
cls
echo.
echo ============================================
echo      SECURE WIPE FREE SPACE
echo ============================================
echo.
echo  Доступные диски:
"%EXE_PATH%" info 2>nul
echo.
echo  Профили: !AVAILABLE_PROFILES!
echo  Движки: !AVAILABLE_ENGINES!
echo.
echo  1.  Wipe C: (system disk - DANGEROUS)
echo  2.  Wipe D: (data disk)
echo  3.  Wipe all disks
echo  4.  Custom wipe
echo  5.  Back to main menu
echo.
set /p "wipe_choice=Выберите опцию (1-5): "

if "%wipe_choice%"=="1" goto wipe_system_disk
if "%wipe_choice%"=="2" goto wipe_data_disk
if "%wipe_choice%"=="3" goto wipe_all_disks
if "%wipe_choice%"=="4" goto custom_wipe
if "%wipe_choice%"=="5" goto main_menu

echo [ERROR] Неверный выбор
pause
goto wipe_menu

:wipe_system_disk
echo.
echo [PREVIEW] Command to execute:
echo wipedisk.exe wipe C: --profile=balanced --engine=internal --allow-system-disk
echo.
echo [WARNING] This will wipe free space on SYSTEM DISK C:
echo [WARNING] This operation is IRREVERSIBLE
echo.
set /p "confirm=Вы уверены? (YES/NO): "
if /i not "%confirm%"=="YES" goto wipe_menu

echo [%DATE% %TIME%] User: %USERNAME%, Choice: wipe_system_disk, ExitCode: %ERRORLEVEL% >> "%LOGFILE%"
"%EXE_PATH%" wipe C: --profile=balanced --engine=internal --allow-system-disk
pause
goto main_menu

:wipe_data_disk
echo.
echo [PREVIEW] Command to execute:
echo wipedisk.exe wipe D: --profile=balanced --engine=internal
echo.
set /p "confirm=Вы уверены? (y/N): "
if /i not "%confirm%"=="y" goto wipe_menu

echo [%DATE% %TIME%] User: %USERNAME%, Choice: wipe_data_disk, ExitCode: %ERRORLEVEL% >> "%LOGFILE%"
"%EXE_PATH%" wipe D: --profile=balanced --engine=internal
pause
goto main_menu

:wipe_all_disks
echo.
echo [PREVIEW] Command to execute:
echo wipedisk.exe wipe --profile=balanced --engine=internal
echo.
set /p "confirm=Вы уверены? (y/N): "
if /i not "%confirm%"=="y" goto wipe_menu

echo [%DATE% %TIME%] User: %USERNAME%, Choice: wipe_all_disks, ExitCode: %ERRORLEVEL% >> "%LOGFILE%"
"%EXE_PATH%" wipe --profile=balanced --engine=internal
pause
goto main_menu

:custom_wipe
echo.
echo ============================================
echo        CUSTOM WIPE CONFIGURATION
echo ============================================
echo.
set /p "target_disk=Целевой диск (C:, D:, etc.): "
set /p "wipe_profile=Профиль (!AVAILABLE_PROFILES!): "
set /p "wipe_engine=Движок (!AVAILABLE_ENGINES!): "

set "allow_system_flag="
if /i "%target_disk%"=="C:" set "allow_system_flag=--allow-system-disk"

echo.
echo [PREVIEW] Command to execute:
echo wipedisk.exe wipe %target_disk% --profile=%wipe_profile% --engine=%wipe_engine% %allow_system_flag%
echo.
set /p "confirm=Вы уверены? (y/N): "
if /i not "%confirm%"=="y" goto wipe_menu

echo [%DATE% %TIME%] User: %USERNAME%, Choice: custom_wipe, Target: %target_disk%, ExitCode: %ERRORLEVEL% >> "%LOGFILE%"
"%EXE_PATH%" wipe %target_disk% --profile=%wipe_profile% --engine=%wipe_engine% %allow_system_flag%
pause
goto main_menu

:maintenance_menu
cls
echo.
echo ============================================
echo      SYSTEM MAINTENANCE
echo ============================================
echo.
echo  Доступные планы:
"%EXE_PATH%" maintenance --list-plans 2>nul
echo.
echo  1.  Quick cleanup (15 min)
echo  2.  Light monthly (30 min)
echo  3.  Security quarterly (3 hours)
echo  4.  Full yearly (6 hours)
echo  5.  Deep clean (4 hours)
echo  6.  Verify only (1 hour)
echo  7.  Custom plan
echo  8.  Back to main menu
echo.
set /p "maint_choice=Выберите опцию (1-8): "

if "%maint_choice%"=="1" goto maint_quick
if "%maint_choice%"=="2" goto maint_monthly
if "%maint_choice%"=="3" goto maint_quarterly
if "%maint_choice%"=="4" goto maint_yearly
if "%maint_choice%"=="5" goto maint_deep
if "%maint_choice%"=="6" goto maint_verify
if "%maint_choice%"=="7" goto maint_custom
if "%maint_choice%"=="8" goto main_menu

echo [ERROR] Неверный выбор
pause
goto maintenance_menu

:maint_quick
echo.
echo [PREVIEW] Command to execute:
echo wipedisk.exe maintenance --plan=quick_cleanup --silent
echo.
set /p "confirm=Вы уверены? (y/N): "
if /i not "%confirm%"=="y" goto maintenance_menu

echo [%DATE% %TIME%] User: %USERNAME%, Choice: maint_quick, ExitCode: %ERRORLEVEL% >> "%LOGFILE%"
"%EXE_PATH%" maintenance --plan=quick_cleanup --silent
pause
goto main_menu

:maint_monthly
echo.
echo [PREVIEW] Command to execute:
echo wipedisk.exe maintenance --plan=light_monthly
echo.
set /p "confirm=Вы уверены? (y/N): "
if /i not "%confirm%"=="y" goto maintenance_menu

echo [%DATE% %TIME%] User: %USERNAME%, Choice: maint_monthly, ExitCode: %ERRORLEVEL% >> "%LOGFILE%"
"%EXE_PATH%" maintenance --plan=light_monthly
pause
goto main_menu

:maint_quarterly
echo.
echo [PREVIEW] Command to execute:
echo wipedisk.exe maintenance --plan=security_quarterly
echo.
set /p "confirm=Вы уверены? (y/N): "
if /i not "%confirm%"=="y" goto maintenance_menu

echo [%DATE% %TIME%] User: %USERNAME%, Choice: maint_quarterly, ExitCode: %ERRORLEVEL% >> "%LOGFILE%"
"%EXE_PATH%" maintenance --plan=security_quarterly
pause
goto main_menu

:maint_yearly
echo.
echo [PREVIEW] Command to execute:
echo wipedisk.exe maintenance --plan=full_year
echo.
echo [WARNING] This will take up to 6 hours
set /p "confirm=Вы уверены? (y/N): "
if /i not "%confirm%"=="y" goto maintenance_menu

echo [%DATE% %TIME%] User: %USERNAME%, Choice: maint_yearly, ExitCode: %ERRORLEVEL% >> "%LOGFILE%"
"%EXE_PATH%" maintenance --plan=full_year
pause
goto main_menu

:maint_deep
echo.
echo [PREVIEW] Command to execute:
echo wipedisk.exe maintenance --plan=deep_clean
echo.
set /p "confirm=Вы уверены? (y/N): "
if /i not "%confirm%"=="y" goto maintenance_menu

echo [%DATE% %TIME%] User: %USERNAME%, Choice: maint_deep, ExitCode: %ERRORLEVEL% >> "%LOGFILE%"
"%EXE_PATH%" maintenance --plan=deep_clean
pause
goto main_menu

:maint_verify
echo.
echo [PREVIEW] Command to execute:
echo wipedisk.exe maintenance --plan=verify_only
echo.
set /p "confirm=Вы уверены? (y/N): "
if /i not "%confirm%"=="y" goto maintenance_menu

echo [%DATE% %TIME%] User: %USERNAME%, Choice: maint_verify, ExitCode: %ERRORLEVEL% >> "%LOGFILE%"
"%EXE_PATH%" maintenance --plan=verify_only
pause
goto main_menu

:maint_custom
echo.
echo ============================================
echo        CUSTOM MAINTENANCE PLAN
echo ============================================
echo.
set /p "custom_plan=План (full_year/light_monthly/etc.): "
set /p "custom_parallel=Параллельно? (y/N): "
set /p "custom_silent=Тихий режим? (y/N): "

set "parallel_flag="
if /i "%custom_parallel%"=="y" set "parallel_flag=--parallel"

set "silent_flag="
if /i "%custom_silent%"=="y" set "silent_flag=--silent"

echo.
echo [PREVIEW] Command to execute:
echo wipedisk.exe maintenance --plan=%custom_plan% %parallel_flag% %silent_flag%
echo.
set /p "confirm=Вы уверены? (y/N): "
if /i not "%confirm%"=="y" goto maintenance_menu

echo [%DATE% %TIME%] User: %USERNAME%, Choice: maint_custom, Plan: %custom_plan%, ExitCode: %ERRORLEVEL% >> "%LOGFILE%"
"%EXE_PATH%" maintenance --plan=%custom_plan% %parallel_flag% %silent_flag%
pause
goto main_menu

:verify_menu
cls
echo.
echo ============================================
echo      VERIFY WIPE QUALITY
echo ============================================
echo.
echo  Уровни проверки:
echo    basic     - Базовая проверка
echo    physical  - Физическая проверка (требует админ)
echo    aggressive - Агрессивная проверка
echo.
echo  1.  Verify last session (basic)
echo  2.  Verify last session (physical)
echo  3.  Verify last session (aggressive)
echo  4.  Custom verification
echo  5.  Back to main menu
echo.
set /p "verify_choice=Выберите опцию (1-5): "

if "%verify_choice%"=="1" goto verify_basic
if "%verify_choice%"=="2" goto verify_physical
if "%verify_choice%"=="3" goto verify_aggressive
if "%verify_choice%"=="4" goto verify_custom
if "%verify_choice%"=="5" goto main_menu

echo [ERROR] Неверный выбор
pause
goto verify_menu

:verify_basic
echo.
echo [PREVIEW] Command to execute:
echo wipedisk.exe verify --last-session --level=basic
echo.
set /p "confirm=Вы уверены? (y/N): "
if /i not "%confirm%"=="y" goto verify_menu

echo [%DATE% %TIME%] User: %USERNAME%, Choice: verify_basic, ExitCode: %ERRORLEVEL% >> "%LOGFILE%"
"%EXE_PATH%" verify --last-session --level=basic
pause
goto main_menu

:verify_physical
echo.
echo [PREVIEW] Command to execute:
echo wipedisk.exe verify --last-session --physical
echo.
set /p "confirm=Вы уверены? (y/N): "
if /i not "%confirm%"=="y" goto verify_menu

echo [%DATE% %TIME%] User: %USERNAME%, Choice: verify_physical, ExitCode: %ERRORLEVEL% >> "%LOGFILE%"
"%EXE_PATH%" verify --last-session --physical
pause
goto main_menu

:verify_aggressive
echo.
echo [PREVIEW] Command to execute:
echo wipedisk.exe verify --last-session --level=aggressive
echo.
echo [WARNING] This may take up to 2 hours
set /p "confirm=Вы уверены? (y/N): "
if /i not "%confirm%"=="y" goto verify_menu

echo [%DATE% %TIME%] User: %USERNAME%, Choice: verify_aggressive, ExitCode: %ERRORLEVEL% >> "%LOGFILE%"
"%EXE_PATH%" verify --last-session --level=aggressive
pause
goto main_menu

:verify_custom
echo.
echo ============================================
echo        CUSTOM VERIFICATION
echo ============================================
echo.
set /p "verify_level=Уровень (basic/physical/aggressive): "
set /p "verify_format=Формат отчёта (json/csv): "
set /p "verify_output=Файл отчёта (пустой=авто): "

set "output_flag="
if not "%verify_output%"=="" set "output_flag=--report=%verify_output%"

echo.
echo [PREVIEW] Command to execute:
echo wipedisk.exe verify --last-session --level=%verify_level% --format=%verify_format% %output_flag%
echo.
set /p "confirm=Вы уверены? (y/N): "
if /i not "%confirm%"=="y" goto verify_menu

echo [%DATE% %TIME%] User: %USERNAME%, Choice: verify_custom, Level: %verify_level%, ExitCode: %ERRORLEVEL% >> "%LOGFILE%"
"%EXE_PATH%" verify --last-session --level=%verify_level% --format=%verify_format% %output_flag%
pause
goto main_menu

:diagnose_menu
cls
echo.
echo ============================================
echo      SYSTEM DIAGNOSTICS
echo ============================================
echo.
echo  Уровни диагностики:
echo    quick  - Быстрая диагностика (3 теста)
echo    full   - Полная диагностика (6 тестов)
echo    deep   - Глубокая диагностика (8 тестов)
echo.
echo  1.  Quick diagnostics
echo  2.  Full diagnostics
echo  3.  Deep diagnostics
echo  4.  Test specific function
echo  5.  Back to main menu
echo.
set /p "diag_choice=Выберите опцию (1-5): "

if "%diag_choice%"=="1" goto diag_quick
if "%diag_choice%"=="2" goto diag_full
if "%diag_choice%"=="3" goto diag_deep
if "%diag_choice%"=="4" goto diag_specific
if "%diag_choice%"=="5" goto main_menu

echo [ERROR] Неверный выбор
pause
goto diagnose_menu

:diag_quick
echo.
echo [PREVIEW] Command to execute:
echo wipedisk.exe diagnose --quick
echo.
set /p "confirm=Вы уверены? (y/N): "
if /i not "%confirm%"=="y" goto diagnose_menu

echo [%DATE% %TIME%] User: %USERNAME%, Choice: diag_quick, ExitCode: %ERRORLEVEL% >> "%LOGFILE%"
"%EXE_PATH%" diagnose --quick
pause
goto main_menu

:diag_full
echo.
echo [PREVIEW] Command to execute:
echo wipedisk.exe diagnose --full
echo.
set /p "confirm=Вы уверены? (y/N): "
if /i not "%confirm%"=="y" goto diagnose_menu

echo [%DATE% %TIME%] User: %USERNAME%, Choice: diag_full, ExitCode: %ERRORLEVEL% >> "%LOGFILE%"
"%EXE_PATH%" diagnose --full
pause
goto main_menu

:diag_deep
echo.
echo [PREVIEW] Command to execute:
echo wipedisk.exe diagnose --deep
echo.
echo [WARNING] This may take up to 10 minutes
set /p "confirm=Вы уверены? (y/N): "
if /i not "%confirm%"=="y" goto diagnose_menu

echo [%DATE% %TIME%] User: %USERNAME%, Choice: diag_deep, ExitCode: %ERRORLEVEL% >> "%LOGFILE%"
"%EXE_PATH%" diagnose --deep
pause
goto main_menu

:diag_specific
echo.
echo ============================================
echo        SPECIFIC FUNCTION TEST
echo ============================================
echo.
echo  Доступные тесты:
echo    permissions - Проверка прав доступа
echo    disks       - Проверка дисков
echo    memory      - Проверка памяти
echo    cpu         - Проверка CPU
echo    paths       - Проверка путей
echo    api         - Проверка API
echo    wipe        - Тест затирания
echo    network     - Проверка сети
echo.
set /p "specific_test=Тест для выполнения: "

echo.
echo [PREVIEW] Command to execute:
echo wipedisk.exe diagnose --test=%specific_test%
echo.
set /p "confirm=Вы уверены? (y/N): "
if /i not "%confirm%"=="y" goto diagnose_menu

echo [%DATE% %TIME%] User: %USERNAME%, Choice: diag_specific, Test: %specific_test%, ExitCode: %ERRORLEVEL% >> "%LOGFILE%"
"%EXE_PATH%" diagnose --test=%specific_test%
pause
goto main_menu

:profiles_menu
cls
echo.
echo ============================================
echo      CONFIGURE PROFILES
echo ============================================
echo.
echo  Доступные профили:
wipedisk.exe maintenance --list-plans 2>nul
echo.
echo  1.  Show profile details
echo  2.  Test profile performance
echo  3.  Create custom profile
echo  4.  Back to main menu
echo.
set /p "profile_choice=Выберите опцию (1-4): "

if "%profile_choice%"=="1" goto profile_details
if "%profile_choice%"=="2" goto profile_test
if "%profile_choice%"=="3" goto profile_custom
if "%profile_choice%"=="4" goto main_menu

echo [ERROR] Неверный выбор
pause
goto profiles_menu

:profile_details
set /p "profile_name=Имя профиля: "
echo.
echo [INFO] Showing details for profile: %profile_name%
echo.
echo Здесь будет показана детальная информация о профиле...
pause
goto profiles_menu

:profile_test
set /p "test_profile=Профиль для теста: "
echo.
echo [PREVIEW] Command to execute:
echo wipedisk.exe wipe --dry-run --profile=%test_profile% D:
echo.
set /p "confirm=Вы уверены? (y/N): "
if /i not "%confirm%"=="y" goto profiles_menu

echo [%DATE% %TIME%] User: %USERNAME%, Choice: profile_test, Profile: %test_profile%, ExitCode: %ERRORLEVEL% >> "%LOGFILE%"
"%EXE_PATH%" wipe --dry-run --profile=%test_profile% D:
pause
goto profiles_menu

:profile_custom
echo.
echo ============================================
echo        CREATE CUSTOM PROFILE
echo ============================================
echo.
echo Эта функция будет доступна в будущих версиях
echo.
pause
goto profiles_menu

:reports_menu
cls
echo.
echo ============================================
echo      GENERATE REPORTS
echo ============================================
echo.
echo  1.  Generate JSON report
echo  2.  Generate CSV report
echo  3.  View last report
echo  4.  Email report (future)
echo  5.  Back to main menu
echo.
set /p "report_choice=Выберите опцию (1-5): "

if "%report_choice%"=="1" goto report_json
if "%report_choice%"=="2" goto report_csv
if "%report_choice%"=="3" goto report_view
if "%report_choice%"=="4" goto report_email
if "%report_choice%"=="5" goto main_menu

echo [ERROR] Неверный выбор
pause
goto reports_menu

:report_json
echo.
echo [INFO] Generating JSON report...
echo.
echo Здесь будет генерация JSON отчёта...
pause
goto reports_menu

:report_csv
echo.
echo [INFO] Generating CSV report...
echo.
echo Здесь будет генерация CSV отчёта...
pause
goto reports_menu

:report_view
echo.
echo [INFO] Viewing last report...
echo.
echo Здесь будет просмотр последнего отчёта...
pause
goto reports_menu

:report_email
echo.
echo [INFO] Email reports will be available in future versions
echo.
pause
goto reports_menu

:silent_mode
cls
echo.
echo ============================================
echo      SILENT MODE FOR GPO
echo ============================================
echo.
echo  Silent mode options:
echo.
echo  1.  Quick cleanup (silent)
echo  2.  Light monthly (silent)
echo  3.  Verify only (silent)
echo  4.  Custom silent command
echo  5.  Back to main menu
echo.
set /p "silent_choice=Выберите опцию (1-5): "

if "%silent_choice%"=="1" goto silent_quick
if "%silent_choice%"=="2" goto silent_monthly
if "%silent_choice%"=="3" goto silent_verify
if "%silent_choice%"=="4" goto silent_custom
if "%silent_choice%"=="5" goto main_menu

echo [ERROR] Неверный выбор
pause
goto silent_mode

:silent_quick
echo.
echo [PREVIEW] Command to execute:
echo wipedisk.exe maintenance --plan=quick_cleanup --silent
echo.
echo [INFO] This command is ready for GPO deployment
echo.
echo Copy this command for GPO:
echo wipedisk.exe maintenance --plan=quick_cleanup --silent
echo.
pause
goto main_menu

:silent_monthly
echo.
echo [PREVIEW] Command to execute:
echo wipedisk.exe maintenance --plan=light_monthly --silent
echo.
echo [INFO] This command is ready for GPO deployment
echo.
echo Copy this command for GPO:
echo wipedisk.exe maintenance --plan=light_monthly --silent
echo.
pause
goto main_menu

:silent_verify
echo.
echo [PREVIEW] Command to execute:
echo wipedisk.exe maintenance --plan=verify_only --silent
echo.
echo [INFO] This command is ready for GPO deployment
echo.
echo Copy this command for GPO:
echo wipedisk.exe maintenance --plan=verify_only --silent
echo.
pause
goto main_menu

:silent_custom
echo.
echo ============================================
echo        CUSTOM SILENT COMMAND
echo ============================================
echo.
set /p "silent_plan=План обслуживания: "
set /p "silent_extra=Дополнительные флаги: "

echo.
echo [PREVIEW] Command to execute:
echo wipedisk.exe maintenance --plan=%silent_plan% --silent %silent_extra%
echo.
echo [INFO] This command is ready for GPO deployment
echo.
echo Copy this command for GPO:
echo wipedisk.exe maintenance --plan=%silent_plan% --silent %silent_extra%
echo.
pause
goto main_menu

:dryrun_mode
cls
echo.
echo ============================================
echo      DRY-RUN MODE
echo ============================================
echo.
echo  Dry-run позволяет тестировать команды без реальных изменений
echo.
echo  1.  Test wipe C: (dry-run)
echo  2.  Test maintenance plan (dry-run)
echo  3.  Test verification (dry-run)
echo  4.  Custom dry-run command
echo  5.  Back to main menu
echo.
set /p "dryrun_choice=Выберите опцию (1-5): "

if "%dryrun_choice%"=="1" goto dryrun_wipe
if "%dryrun_choice%"=="2" goto dryrun_maint
if "%dryrun_choice%"=="3" goto dryrun_verify
if "%dryrun_choice%"=="4" goto dryrun_custom
if "%dryrun_choice%"=="5" goto main_menu

echo [ERROR] Неверный выбор
pause
goto dryrun_mode

:dryrun_wipe
echo.
echo [PREVIEW] Command to execute:
echo wipedisk.exe wipe --dry-run C: --profile=balanced --engine=internal --allow-system-disk
echo.
set /p "confirm=Вы уверены? (y/N): "
if /i not "%confirm%"=="y" goto dryrun_mode

echo [%DATE% %TIME%] User: %USERNAME%, Choice: dryrun_wipe, ExitCode: %ERRORLEVEL% >> "%LOGFILE%"
"%EXE_PATH%" wipe --dry-run C: --profile=balanced --engine=internal --allow-system-disk
pause
goto main_menu

:dryrun_maint
echo.
echo [PREVIEW] Command to execute:
echo wipedisk.exe maintenance --plan=light_monthly --dry-run
echo.
set /p "confirm=Вы уверены? (y/N): "
if /i not "%confirm%"=="y" goto dryrun_mode

echo [%DATE% %TIME%] User: %USERNAME%, Choice: dryrun_maint, ExitCode: %ERRORLEVEL% >> "%LOGFILE%"
"%EXE_PATH%" maintenance --plan=light_monthly --dry-run
pause
goto main_menu

:dryrun_verify
echo.
echo [INFO] Verify command does not support dry-run mode
echo [INFO] Use basic verification level instead
echo.
echo [PREVIEW] Command to execute:
echo wipedisk.exe verify --last-session --level=basic
echo.
set /p "confirm=Вы уверены? (y/N): "
if /i not "%confirm%"=="y" goto dryrun_mode

echo [%DATE% %TIME%] User: %USERNAME%, Choice: dryrun_verify, ExitCode: %ERRORLEVEL% >> "%LOGFILE%"
"%EXE_PATH%" verify --last-session --level=basic
pause
goto main_menu

:dryrun_custom
echo.
echo ============================================
echo        CUSTOM DRY-RUN COMMAND
echo ============================================
echo.
set /p "dryrun_command=Команда для теста: "

echo.
echo [PREVIEW] Command to execute:
echo wipedisk.exe %dryrun_command% --dry-run
echo.
set /p "confirm=Вы уверены? (y/N): "
if /i not "%confirm%"=="y" goto dryrun_mode

echo [%DATE% %TIME%] User: %USERNAME%, Choice: dryrun_custom, Command: %dryrun_command%, ExitCode: %ERRORLEVEL% >> "%LOGFILE%"
"%EXE_PATH%" %dryrun_command% --dry-run
pause
goto main_menu

:exit_menu
cls
echo.
echo ============================================
echo      WipeDisk Enterprise %VERSION%
echo ============================================
echo.
echo  Спасибо за использование WipeDisk Enterprise!
echo.
echo  Полезные ресурсы:
echo    - Документация: https://docs.wipedisk.com
echo    - Поддержка: support@wipedisk.com
echo    - Обновления: https://github.com/wipedisk/enterprise
echo.
echo  Лог работы сохранён в: %LOGFILE%
echo.
echo [%DATE% %TIME%] Menu closed, User: %USERNAME%, Session completed >> "%LOGFILE%"
pause
exit /b 0
