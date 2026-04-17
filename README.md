# 🎵 YouTube Downloader

A portable and modern Go application to download songs from YouTube, YouTube Music, or search by name directly at **128kbps MP3**.

## ✨ Features

- **Modern dark UI** (Fyne v2)
- **Parallel downloads** - up to 3 songs simultaneously (configurable)
- **Multiline text input** - paste multiple URLs/names, one per line
- **Progress bar** showing percentage of songs completed
- **Real-time logs** with timestamps and per-song tags
- **Native Windows folder picker** (comfortable and large)
- **Auto-download dependencies** - yt-dlp and ffmpeg are downloaded automatically on first run
- **Portable** - single executable, no installation required
- **Cross-platform** - code compatible with Windows, Linux and macOS

## 🚀 Quick Start

### Requirements
- **Windows 10/11** (tested on both versions)
- **Go 1.21+** (only if compiling from source)
- **GCC/MinGW** (for compiling, MSYS2 recommended)

### Using the Precompiled Executable

1. Download `YouTubeDownloader.exe` from this repository
2. Run the `.exe`
   - **First time:** Will automatically download `yt-dlp.exe` (~10MB) and `ffmpeg.exe` (~8MB) in the same folder
   - Songs are saved in the `Musica/` folder next to the `.exe`
3. Ready to download!

### Compiling from Source

#### Prerequisites

1. **Install Go**
   ```bash
   # Download from https://golang.org/dl/
   # Verify installation
   go version
   ```

2. **Install MSYS2 + GCC** (if you don't have a compiler)
   ```bash
   # Download MSYS2 from https://www.msys2.org/
   # During installation, select the mingw64 installation
   # After installing, open MSYS2 and run:
   pacman -S mingw-w64-ucrt-x86_64-gcc
   ```

3. **Clone/Download the Repository**
   ```bash
   git clone https://github.com/yourusername/YoutubeDownloader.git
   cd YoutubeDownloader
   ```

#### Compilation

```bash
# Option 1: In Windows PowerShell (recommended)
$env:PATH = "C:\msys64\ucrt64\bin;$env:PATH"
go build -ldflags="-H windowsgui -s -w" -o YouTubeDownloader.exe .

# Option 2: Run the build.bat file (if in repository)
.\build.bat
```

**Flag explanation:**
- `-H windowsgui` - Hides the console window (shows only the Fyne UI)
- `-s -w` - Minifies the executable (no debug info)

## 📖 How to Use

### Interface

```
┌─────────────────────────────────────┐
│   🎵 YouTube Downloader             │
├─────────────────────────────────────┤
│ [Multiline text box]                │
│  Paste URLs or names here...        │
│                                     │
│ 📁 ~/Music/  [Change folder]        │
├─────────────────────────────────────┤
│ [Download real-time log]            │
│ [15:04:05] ✅ yt-dlp ready.         │
│ [15:04:06] 📋 Starting 3 songs      │
│ [15:04:07] [1/3] ▶ Bohemian...      │
│ ...                                 │
├─────────────────────────────────────┤
│ Progress: [===========     ] 2 / 3   │
└─────────────────────────────────────┘
```

### Input Examples

You can mix URLs and song names:

```
https://www.youtube.com/watch?v=dQw4w9WgXcQ
Bohemian Rhapsody Queen
https://music.youtube.com/watch?v=abcd1234
Hotel California Eagles
Shape of You Ed Sheeran
```

Press **"⬇ Download"** and all 5 songs will download in parallel (max 3 simultaneous).

### Change Download Folder

1. Click **"Change folder"**
2. The native Windows explorer opens
3. Select a folder and press OK
4. Future downloads will go to that folder

## 🏗️ Project Structure

```
YoutubeDownloader/
├── main.go                 # Main code (UI + download logic)
├── zip_extract.go          # ffmpeg.zip extractor
├── sysproc_windows.go      # Windows-specific console hiding
├── sysproc_other.go        # macOS/Linux no-op
├── go.mod                  # Go modules
├── go.sum                  # Dependency checksums
├── build.bat               # Windows build script
├── YouTubeDownloader.exe   # Compiled executable
├── README.md               # This file
└── Musica/                 # Downloads folder (created automatically)
```

## 📦 Dependencies

### External (Code)
- **fyne.io/fyne/v2** - Cross-platform UI framework

### External (Executables)
Downloaded automatically:
- **yt-dlp.exe** - YouTube downloader (GitHub: yt-dlp/yt-dlp)
- **ffmpeg.exe** - Audio converter (GitHub: GyanD/codexffmpeg - essentials build)

### Go Libraries (Automatic)
Installed with `go mod tidy`:
- `golang.org/x/text` - Language support
- `golang.org/x/image` - Image processing
- `github.com/go-gl/glfw/v3.3/glfw` - OpenGL
- And many more... (handled automatically)

## ⚙️ Advanced Configuration

### Change Maximum Parallel Downloads

Edit `main.go` line ~24:

```go
const maxConcurrent = 3  // Change this number
```

**Recommendations:**
- `2-3` - For normal connections
- `5+` - If you have fiber and YouTube doesn't block you
- `1` - If you have connection issues

### Change Audio Quality

Edit `main.go` in the `downloadOne()` function, find:

```go
"--audio-quality", "128K",  // Change 128K to 192K, 256K, etc.
```

**Options:** `128K`, `192K`, `256K`, `320K` (maximum)

### Change Default Folder

Edit `main.go` line ~36:

```go
outputDir = filepath.Join(appDir, "Musica")  // Change "Musica" to your preference
```

## 🐛 Troubleshooting

### "❌ Error downloading yt-dlp"
- Verify you have internet connection
- Your firewall might be blocking downloads
- Try downloading manually from: https://github.com/yt-dlp/yt-dlp/releases

### "⚠ Error with ffmpeg"
- Same as above, verify connection
- Or download manually from: https://github.com/GyanD/codexffmpeg/releases
- Place `ffmpeg.exe` in the same folder as the app

### "yt-dlp not found"
- Make sure `yt-dlp.exe` is in the same folder as `YouTubeDownloader.exe`
- Try running manually: `yt-dlp.exe --version`

### Download fails with "HTTP 429"
- YouTube temporarily blocked you for too many downloads
- Wait 1-2 hours or reduce `maxConcurrent` to 1
- Try using VPN (yt-dlp supports proxies)

### Can't download from YouTube Music
- Some YT Music videos require authentication
- Try searching by name instead: `"Song Name Artist"`

## 💡 Usage Tips

1. **Spotify Playlist → YouTube** - Use an online converter to get YouTube URLs, then copy them to the app
2. **Search by name** - `"All Too Well Taylor Swift"` works better than cryptic URLs
3. **Mix formats** - You can mix URLs and names without any problem
4. **Rename later** - MP3s are saved with the YouTube title, you can rename them afterwards
5. **Keep .exe with ffmpeg and yt-dlp** - Don't move the app to another folder without the dependencies

## 🔒 Privacy

- This app is **offline** (except when downloading yt-dlp/ffmpeg on first run)
- Only connects to:
  - GitHub (to download yt-dlp/ffmpeg)
  - YouTube (to download videos)
- No telemetry or tracking is sent

## 📄 License

MIT - Feel free to use, modify, and distribute

## 👨‍💻 Development

### Debug Build

```bash
go build -o YouTubeDownloader.exe .  # Without -ldflags for debugging
```

### Cross-compile to Linux

```bash
GOOS=linux GOARCH=amd64 go build -o YouTubeDownloader-linux .
```

### View logs in terminal

For debugging, edit `main.go` and remove `-H windowsgui` from ldflags before compiling.

## 🤝 Contributions

Pull requests welcome. For large changes, open an issue first.

## 📝 Changelog

### v1.0 - Initial
- ✅ Download songs from YouTube at 128kbps MP3
- ✅ Parallel downloads (max 3)
- ✅ Modern UI with Fyne
- ✅ Auto-download dependencies
- ✅ Native Windows folder picker

---

**Made with ❤️ in Go**

Questions? Open an issue on GitHub or contact directly.

