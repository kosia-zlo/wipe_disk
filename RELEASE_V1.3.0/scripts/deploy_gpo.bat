@echo off
chcp 65001 >nul
setlocal enabledelayedexpansion

:: WipeDisk Enterprise 1.2.2 GPO Deployment Script
:: ===================================================

set "WIPE_VERSION=1.2.2"
set "TITLE=WipeDisk Enterprise GPO Deployment"
set "LOGFILE=%TEMP%\wipedisk_gpo_deploy.log"

:: Проверка админ прав
net session >nul 2>&1
if %errorlevel% neq 0 (
    echo [ERROR] Требуются права администратора для GPO развертывания
    pause
    exit /b 1
)

echo.
echo ============================================
echo   WipeDisk Enterprise %WIPE_VERSION% GPO Deployment
echo ============================================
echo.

:: Проверка наличия WipeDisk
if not exist "wipedisk.exe" (
    echo [ERROR] wipedisk.exe не найден в текущей директории
    echo Пожалуйста, поместите wipedisk.exe в эту директорию
    pause
    exit /b 1
)

:: Определение путей установки
set "INSTALL_DIR=C:\Program Files\WipeDisk Enterprise"
set "MENU_DIR=%INSTALL_DIR\Menu"
set "LOG_DIR=%INSTALL_DIR\Logs"

echo [INFO] Директория установки: %INSTALL_DIR%
echo [INFO] Директория меню: %MENU_DIR%
echo [INFO] Директория логов: %LOG_DIR%
echo.

:: Создание директорий
echo [INFO] Создание директорий...
if not exist "%INSTALL_DIR%" mkdir "%INSTALL_DIR%"
if not exist "%MENU_DIR%" mkdir "%MENU_DIR%"
if not exist "%LOG_DIR%" mkdir "%LOG_DIR%"

:: Копирование файлов
echo [INFO] Копирование файлов...
copy "wipedisk.exe" "%INSTALL_DIR%\" >nul
if exist "wipedisk_menu.bat" copy "wipedisk_menu.bat" "%MENU_DIR%\" >nul

:: Создание ярлыков
echo [INFO] Создание ярлыков...
powershell -Command "$WshShell = New-Object -comObject WScript.Shell; $Shortcut = $WshShell.CreateShortcut('%PUBLIC%\Desktop\WipeDisk Enterprise.lnk'); $Shortcut.TargetPath = '%MENU_DIR%\wipedisk_menu.bat'; $Shortcut.Save()" >nul

powershell -Command "$WshShell = New-Object -comObject WScript.Shell; $Shortcut = $WshShell.CreateShortcut('%PROGRAMDATA%\Microsoft\Windows\Start Menu\Programs\WipeDisk Enterprise.lnk'); $Shortcut.TargetPath = '%MENU_DIR%\wipedisk_menu.bat'; $Shortcut.Save()" >nul

:: Настройка переменных окружения
echo [INFO] Настройка переменных окружения...
setx WIPEDISK_HOME "%INSTALL_DIR%" /M >nul
setx WIPEDISK_LOGS "%LOG_DIR%" /M >nul

:: Создание GPO шаблонов
echo [INFO] Создание GPO шаблонов...
call :create_gpo_templates

:: Настройка прав доступа
echo [INFO] Настройка прав доступа...
icacls "%INSTALL_DIR%" /grant "Administrators:(OI)(CI)F" /T >nul
icacls "%INSTALL_DIR%" /grant "SYSTEM:(OI)(CI)F" /T >nul
icacls "%LOG_DIR%" /grant "Users:(OI)(CI)M" /T >nul

:: Создание scheduled tasks для автоматического обслуживания
echo [INFO] Создание scheduled tasks...
call :create_scheduled_tasks

:: Тестирование установки
echo [INFO] Тестирование установки...
"%INSTALL_DIR%\wipedisk.exe" --version >nul 2>&1
if %errorlevel% neq 0 (
    echo [ERROR] Тестирование wipedisk.exe не удалось
    pause
    exit /b 1
)

echo.
echo ============================================
echo        УСПЕШНОЕ РАЗВЕРТЫВАНИЕ
echo ============================================
echo.
echo  WipeDisk Enterprise %WIPE_VERSION% успешно установлен
echo.
echo  Пути:
echo    Исполняемый файл: %INSTALL_DIR%\wipedisk.exe
echo    Меню: %MENU_DIR%\wipedisk_menu.bat
echo    Логи: %LOG_DIR%
echo.
echo  Ярлыки созданы:
echo    На рабочем столе всех пользователей
echo    в меню Пуск всех пользователей
echo.
echo  GPO шаблоны готовы для развертывания
echo.
echo  Следующие шаги:
echo    1. Откройте Group Policy Management Console
echo    2. Импортируйте шаблоны из %INSTALL_DIR%\GPO\
echo    3. Настройте политики для нужных OU
echo    4. Протестируйте на тестовой машине
echo.

pause
exit /b 0

:create_gpo_templates
set "GPO_DIR=%INSTALL_DIR%\GPO"
if not exist "%GPO_DIR%" mkdir "%GPO_DIR%"

:: Создание шаблона для быстрой очистки
(
echo @echo off
echo chcp 65001 ^>nul
echo.
echo echo WipeDisk Enterprise - Quick Cleanup
echo echo ==================================
echo.
echo "%INSTALL_DIR%\wipedisk.exe" maintenance --plan=quick_cleanup --silent
echo.
echo if %%errorlevel%% equ 0 (
echo     echo [SUCCESS] Quick cleanup completed
echo ) else (
echo     echo [ERROR] Quick cleanup failed with code %%errorlevel%%
echo )
echo.
echo exit /b %%errorlevel%%
) > "%GPO_DIR%\quick_cleanup.bat"

:: Создание шаблона для ежемесячного обслуживания
(
echo @echo off
echo chcp 65001 ^>nul
echo.
echo echo WipeDisk Enterprise - Monthly Maintenance
echo echo ========================================
echo.
echo "%INSTALL_DIR%\wipedisk.exe" maintenance --plan=light_monthly --silent
echo.
echo if %%errorlevel%% equ 0 (
echo     echo [SUCCESS] Monthly maintenance completed
echo ) else (
echo     echo [ERROR] Monthly maintenance failed with code %%errorlevel%%
echo )
echo.
echo exit /b %%errorlevel%%
) > "%GPO_DIR%\monthly_maintenance.bat"

:: Создание шаблона для верификации
(
echo @echo off
echo chcp 65001 ^>nul
echo.
echo echo WipeDisk Enterprise - Verification
echo echo ==============================
echo.
echo "%INSTALL_DIR%\wipedisk.exe" verify --last-session --level=basic --report="%LOG_DIR%\verify_%%date:~-10,4%%date:~-7,2%%date:~-4,2%.json"
echo.
echo if %%errorlevel%% equ 0 (
echo     echo [SUCCESS] Verification completed
echo ) else (
echo     echo [ERROR] Verification failed with code %%errorlevel%%
echo )
echo.
echo exit /b %%errorlevel%%
) > "%GPO_DIR%\verification.bat"

:: Создание PowerShell скрипта для GPO
(
echo # WipeDisk Enterprise GPO Deployment Script
echo # Version %WIPE_VERSION%
echo.
echo Write-Host "WipeDisk Enterprise GPO Deployment" -ForegroundColor Green
echo Write-Host "================================" -ForegroundColor Green
echo.
echo # Check admin rights
echo $currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
echo $principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
echo if (-not $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
echo     Write-Host "ERROR: Administrator rights required" -ForegroundColor Red
echo     exit 1
echo }
echo.
echo # Installation paths
echo $installDir = "%INSTALL_DIR%"
echo $menuDir = "%MENU_DIR%"
echo $logDir = "%LOG_DIR%"
echo.
echo # Create directories
echo New-Item -ItemType Directory -Force -Path $installDir | Out-Null
echo New-Item -ItemType Directory -Force -Path $menuDir | Out-Null
echo New-Item -ItemType Directory -Force -Path $logDir | Out-Null
echo.
echo # Copy files
echo Copy-Item "wipedisk.exe" $installDir -Force
echo if (Test-Path "wipedisk_menu.bat") {
echo     Copy-Item "wipedisk_menu.bat" $menuDir -Force
echo }
echo.
echo # Create shortcuts
echo $WshShell = New-Object -ComObject WScript.Shell
echo $AllUsersDesktop = [Environment]::GetFolderPath("CommonDesktopDirectory")
echo $AllUsersStartMenu = [Environment]::GetFolderPath("CommonStartMenu")
echo.
echo $desktopShortcut = $WshShell.CreateShortcut("$AllUsersDesktop\WipeDisk Enterprise.lnk")
echo $desktopShortcut.TargetPath = "$menuDir\wipedisk_menu.bat"
echo $desktopShortcut.Save()
echo.
echo $startMenuShortcut = $WshShell.CreateShortcut("$AllUsersStartMenu\Programs\WipeDisk Enterprise.lnk")
echo $startMenuShortcut.TargetPath = "$menuDir\wipedisk_menu.bat"
echo $startMenuShortcut.Save()
echo.
echo # Set environment variables
echo [Environment]::SetEnvironmentVariable("WIPEDISK_HOME", $installDir, "Machine")
echo [Environment]::SetEnvironmentVariable("WIPEDISK_LOGS", $logDir, "Machine")
echo.
echo # Set permissions
echo $acl = Get-Acl $installDir
echo $accessRule = New-Object System.Security.AccessControl.FileSystemAccessRule("Administrators", "FullControl", "ContainerInherit,ObjectInherit", "None", "Allow")
echo $acl.SetAccessRule($accessRule)
echo Set-Acl $installDir $acl
echo.
echo Write-Host "Installation completed successfully!" -ForegroundColor Green
echo Write-Host "Executable: $installDir\wipedisk.exe" -ForegroundColor Cyan
echo Write-Host "Menu: $menuDir\wipedisk_menu.bat" -ForegroundColor Cyan
echo Write-Host "Logs: $logDir" -ForegroundColor Cyan
) > "%GPO_DIR%\deploy_gpo.ps1"

echo [INFO] GPO шаблоны созданы в %GPO_DIR%
goto :eof

:create_scheduled_tasks
echo [INFO] Создание scheduled tasks...

:: Ежедневная быстрая очистка (в 2:00 ночи)
schtasks /create /tn "WipeDisk\QuickCleanup" /tr "\"%INSTALL_DIR%\wipedisk.exe\" maintenance --plan=quick_cleanup --silent" /sc daily /st 02:00 /ru "SYSTEM" /f >nul 2>&1

:: Ежемесячное обслуживание (1-го числа в 3:00 ночи)
schtasks /create /tn "WipeDisk\MonthlyMaintenance" /tr "\"%INSTALL_DIR%\wipedisk.exe\" maintenance --plan=light_monthly --silent" /sc monthly /d 1 /st 03:00 /ru "SYSTEM" /f >nul 2>&1

:: Еженедельная верификация (воскресенье в 4:00 утра)
schtasks /create /tn "WipeDisk\WeeklyVerification" /tr "\"%INSTALL_DIR%\wipedisk.exe\" verify --last-session --level=basic --report=\"%LOG_DIR%\verify_%%date:~-10,4%%date:~-7,2%%date:~-4,2%.json\"" /sc weekly /d SUN /st 04:00 /ru "SYSTEM" /f >nul 2>&1

echo [INFO] Scheduled tasks созданы
goto :eof
