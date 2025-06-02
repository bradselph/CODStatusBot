@echo off
setlocal enabledelayedexpansion
REM Root directory - current folder by default
set "ROOT_DIR=%cd%"
REM Find all .go files except *_test.go and run gotests -all -w on each
for /r "%ROOT_DIR%" %%f in (*.go) do (
    set "filename=%%~nxf"
    REM Skip test files
    echo %%f | findstr /i "_test.go" >nul
    if errorlevel 1 (
        echo Generating tests for %%f
        gotests -all -w "%%f"
    )
)
echo All test files generated.
exit /b 0
