#!/bin/bash
# Start Chrome/Chromium with remote debugging enabled for chromedp transport

set -e

# Default port
PORT=${1:-9222}

# Detect OS and find Chrome binary
detect_chrome() {
    if [[ "$OSTYPE" == "darwin"* ]]; then
        # macOS
        if [ -x "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome" ]; then
            echo "/Applications/Google Chrome.app/Contents/MacOS/Google Chrome"
        elif [ -x "/Applications/Chromium.app/Contents/MacOS/Chromium" ]; then
            echo "/Applications/Chromium.app/Contents/MacOS/Chromium"
        fi
    elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
        # Linux
        if command -v google-chrome &> /dev/null; then
            echo "google-chrome"
        elif command -v chromium-browser &> /dev/null; then
            echo "chromium-browser"
        elif command -v chromium &> /dev/null; then
            echo "chromium"
        fi
    elif [[ "$OSTYPE" == "msys" ]] || [[ "$OSTYPE" == "cygwin" ]]; then
        # Windows
        if [ -x "C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe" ]; then
            echo "C:\\Program Files\\Google\\Chrome\\Application\\chrome.exe"
        elif [ -x "C:\\Program Files (x86)\\Google\\Chrome\\Application\\chrome.exe" ]; then
            echo "C:\\Program Files (x86)\\Google\\Chrome\\Application\\chrome.exe"
        fi
    fi
}

# Find Chrome
CHROME=$(detect_chrome)

if [ -z "$CHROME" ]; then
    echo "❌ Chrome/Chromium not found"
    echo ""
    echo "Install Chrome:"
    echo "  macOS:   brew install google-chrome"
    echo "  Linux:   sudo apt-get install chromium-browser"
    echo "  Windows: https://www.google.com/chrome/"
    exit 1
fi

echo "Starting Chrome on port $PORT..."
echo "Chrome binary: $CHROME"
echo ""
echo "DevTools will be available at: http://localhost:$PORT"
echo "Press Ctrl+C to stop Chrome"
echo ""

# Start Chrome with debugging enabled
"$CHROME" \
  --headless \
  --disable-gpu \
  --remote-debugging-port=$PORT \
  --no-sandbox \
  --disable-dev-shm-usage \
  --disable-background-networking \
  --disable-client-side-phishing-detection \
  --disable-component-extensions-with-background-pages
