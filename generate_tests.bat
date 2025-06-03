@echo off
setlocal enabledelayedexpansion

REM Root directory - current folder by default
set "ROOT_DIR=%cd%"
set "TEST_DIR=%ROOT_DIR%\tests"

REM Create test directory if it doesn't exist
if not exist "%TEST_DIR%" (
    mkdir "%TEST_DIR%"
)

REM Loop through all .go files except *_test.go
for /r "%ROOT_DIR%" %%f in (*.go) do (
    echo %%f | findstr /i "_test.go" >nul
    if errorlevel 1 (
        echo Generating tests for %%f
        gotests -all -w %%f
    )
)

REM Move all *_test.go files to the tests directory
for /r "%ROOT_DIR%" %%t in (*_test.go) do (
    move /Y "%%~ft" "%TEST_DIR%\%%~nxt" >nul
)

echo All test files generated and moved to \tests.
exit /b 0
