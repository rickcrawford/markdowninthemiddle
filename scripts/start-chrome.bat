@echo off
REM Start Chrome with remote debugging enabled for chromedp transport

setlocal enabledelayedexpansion

REM Default port
set PORT=9222
if not "%1"=="" set PORT=%1

REM Try to find Chrome in common Windows locations
set CHROME=
if exist "C:\Program Files\Google\Chrome\Application\chrome.exe" (
    set CHROME=C:\Program Files\Google\Chrome\Application\chrome.exe
) else if exist "C:\Program Files (x86)\Google\Chrome\Application\chrome.exe" (
    set CHROME=C:\Program Files (x86)\Google\Chrome\Application\chrome.exe
) else if exist "C:\Program Files\Chromium\Application\chrome.exe" (
    set CHROME=C:\Program Files\Chromium\Application\chrome.exe
)

if "!CHROME!"=="" (
    echo Error: Chrome/Chromium not found
    echo.
    echo Install Chrome from: https://www.google.com/chrome/
    exit /b 1
)

echo Starting Chrome on port %PORT%...
echo Chrome binary: !CHROME!
echo.
echo DevTools will be available at: http://localhost:%PORT%
echo Press Ctrl+C to stop Chrome
echo.

REM Start Chrome with debugging enabled
"!CHROME!" ^
  --headless ^
  --disable-gpu ^
  --remote-debugging-port=%PORT% ^
  --no-sandbox ^
  --disable-dev-shm-usage ^
  --disable-background-networking ^
  --disable-client-side-phishing-detection ^
  --disable-component-extensions-with-background-pages

pause
