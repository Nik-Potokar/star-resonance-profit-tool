# README

Running requires administrator mode

Data collection requires the game to be at 1080p resolution with the interface at the trading center

Data file is stored in the user's home directory as xhgm_prices.json, you can modify the corresponding output and recipes in /good/templates/items.json

## Dependencies
[golang](https://go.dev/)

[wails](https://wails.io/zh-Hans/docs/gettingstarted/installation)

## Running the Code

go get github.com/go-vgo/robotgo (first run)

go mod tidy (first run)


wails dev

## Build Commands
### windows
wails build -clean -o srpt.exe

### Cross-platform
GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc CXX=x86_64-w64-mingw32-g++ wails build -clean -o srpt.exe -platform windows/amd64