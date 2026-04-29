@echo off
setlocal

set "SCRIPT_DIR=%~dp0"
for %%I in ("%SCRIPT_DIR%..") do set "REPO_ROOT=%%~fI"

echo This operation removes all admin roles and triggers bootstrap behavior.
set /p "CONFIRM_TOKEN=Type RESET-ADMINS to continue (or anything else to cancel): "

if /I not "%CONFIRM_TOKEN%"=="RESET-ADMINS" (
    echo Cancelled. Confirmation token did not match.
    exit /b 2
)

echo Running fallback reset from "%REPO_ROOT%"...
pushd "%REPO_ROOT%" >nul
go run .\scripts\fallback_reset_admins.go --confirm RESET-ADMINS
set "EXIT_CODE=%ERRORLEVEL%"
popd >nul

if not "%EXIT_CODE%"=="0" (
    echo Fallback reset failed with exit code %EXIT_CODE%.
    exit /b %EXIT_CODE%
)

echo Fallback reset completed successfully.
exit /b 0
