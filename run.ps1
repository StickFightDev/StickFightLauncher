cls
# Please use ./build.bat when creating release builds!

go build -ldflags="-s -w" -o StickFightLauncher.exe; .\StickFightLauncher.exe
