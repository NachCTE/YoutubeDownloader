@echo off
echo Compilando YouTube Downloader...
set PATH=C:\msys64\ucrt64\bin;%PATH%
go build -ldflags="-H windowsgui -s -w" -o YouTubeDownloader.exe .
if %ERRORLEVEL% == 0 (
    echo.
    echo BUILD EXITOSO: YouTubeDownloader.exe
) else (
    echo.
    echo ERROR en la compilacion.
)
pause

