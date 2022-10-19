@echo off
:: Please use govvv when creating release builds! (go install github.com/JoshuaDoes/govvv@latest)

govvv build -ldflags="-s -w" -o StickFightLauncher.exe
:: go build -ldflags="-s -w" -o StickFightLauncher.exe
