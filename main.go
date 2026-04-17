package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const maxConcurrent = 3

var (
	overallVal  = binding.NewFloat()
	logText     = binding.NewString()
	statusText  = binding.NewString()
	logMu       sync.Mutex
	downloading = false
	appDir      string
	outputDir   string
)

func main() {
	ex, _ := os.Executable()
	appDir = filepath.Dir(ex)
	outputDir = filepath.Join(appDir, "Musica")
	os.MkdirAll(outputDir, 0755)

	a := app.New()
	a.Settings().SetTheme(theme.DarkTheme())

	w := a.NewWindow("🎵 YouTube Downloader")
	w.Resize(fyne.NewSize(720, 540))
	w.SetFixedSize(false)

	// --- Widgets ---
	urlEntry := widget.NewMultiLineEntry()
	urlEntry.SetPlaceHolder("Pegá URLs o nombres de canciones, una por línea...\n\nEjemplo:\nhttps://www.youtube.com/watch?v=...\nBohemian Rhapsody Queen\nhttps://music.youtube.com/watch?v=...")
	urlEntry.Wrapping = fyne.TextWrapWord

	entryScroll := container.NewScroll(urlEntry)
	entryScroll.SetMinSize(fyne.NewSize(680, 120))

	overallBar := widget.NewProgressBarWithData(overallVal)
	overallBar.Min = 0
	overallBar.Max = 100

	logLabel := widget.NewLabelWithData(logText)
	logLabel.Wrapping = fyne.TextWrapWord
	logLabel.TextStyle = fyne.TextStyle{Monospace: true}

	logScroll := container.NewScroll(logLabel)
	logScroll.SetMinSize(fyne.NewSize(680, 210))

	statusLabel := widget.NewLabelWithData(statusText)
	statusLabel.TextStyle = fyne.TextStyle{Bold: true}

	outputLabel := widget.NewLabel("📁 " + shortPath(outputDir))

	outputBtn := widget.NewButton("Cambiar carpeta", func() {
		go func() {
			path, err := pickFolderWindows()
			if err != nil || path == "" {
				return
			}
			outputDir = path
			outputLabel.SetText("📁 " + shortPath(path))
		}()
	})
	outputBtn.Importance = widget.LowImportance

	var downloadBtn *widget.Button
	downloadBtn = widget.NewButton("⬇  Descargar", func() {
		raw := strings.TrimSpace(urlEntry.Text)
		if raw == "" {
			appendLog("⚠  Ingresá al menos una URL o nombre de canción.")
			return
		}
		if downloading {
			return
		}
		var queries []string
		for _, line := range strings.Split(raw, "\n") {
			q := strings.TrimSpace(line)
			if q != "" {
				queries = append(queries, q)
			}
		}
		downloadBtn.Disable()
		go func() {
			startBatchDownload(queries)
			downloadBtn.Enable()
		}()
	})
	downloadBtn.Importance = widget.HighImportance

	// Progress row
	progressRow := container.NewBorder(nil, nil, widget.NewLabel("Progreso:"), statusLabel, overallBar)

	// Layout
	topSection := container.NewVBox(
		entryScroll,
		container.NewBorder(nil, nil,
			container.NewHBox(outputLabel, outputBtn),
			downloadBtn,
			nil,
		),
		widget.NewSeparator(),
	)
	bottomSection := container.NewVBox(
		widget.NewSeparator(),
		progressRow,
	)

	content := container.NewBorder(
		topSection,
		bottomSection,
		nil, nil,
		container.NewPadded(logScroll),
	)

	w.SetContent(content)

	go func() {
		appendLog("🔍 Verificando dependencias...")
		if err := ensureYtDlp(); err != nil {
			appendLog("❌ Error descargando yt-dlp: " + err.Error())
		} else {
			appendLog("✅ yt-dlp listo.")
		}
		if err := ensureFFmpeg(); err != nil {
			appendLog("❌ Error con ffmpeg: " + err.Error())
		} else {
			appendLog("✅ ffmpeg listo.")
		}
		appendLog(fmt.Sprintf("🎵 Listo. Descarga paralela (máx %d simultáneas). Ingresá canciones y presioná Descargar.", maxConcurrent))
	}()

	w.ShowAndRun()
}

// ─── Windows native folder picker ───────────────────────────────────────────

func pickFolderWindows() (string, error) {
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command",
		`Add-Type -AssemblyName System.Windows.Forms;`+
			`$d = New-Object System.Windows.Forms.FolderBrowserDialog;`+
			`$d.Description = 'Seleccionar carpeta de destino';`+
			`$d.ShowNewFolderButton = $true;`+
			`if ($d.ShowDialog() -eq 'OK') { Write-Output $d.SelectedPath }`)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// ─── Logging ────────────────────────────────────────────────────────────────

func appendLog(msg string) {
	logMu.Lock()
	defer logMu.Unlock()
	cur, _ := logText.Get()
	ts := time.Now().Format("15:04:05")
	newVal := cur + fmt.Sprintf("[%s] %s\n", ts, msg)
	lines := strings.Split(newVal, "\n")
	if len(lines) > 300 {
		lines = lines[len(lines)-300:]
	}
	logText.Set(strings.Join(lines, "\n"))
}

// ─── Batch parallel download ─────────────────────────────────────────────────

var progressRegex = regexp.MustCompile(`\[download\]\s+([\d.]+)%`)

func startBatchDownload(queries []string) {
	downloading = true
	total := len(queries)
	overallVal.Set(0)
	statusText.Set(fmt.Sprintf("0 / %d", total))
	appendLog(fmt.Sprintf("📋 Iniciando descarga de %d canción(es) (máx %d en paralelo).", total, maxConcurrent))

	var completed int64
	var okCount int64
	sem := make(chan struct{}, maxConcurrent)
	var wg sync.WaitGroup

	for i, q := range queries {
		wg.Add(1)
		go func(idx int, query string) {
			defer wg.Done()
			sem <- struct{}{}        // acquire
			defer func() { <-sem }() // release

			tag := fmt.Sprintf("[%d/%d]", idx+1, total)
			appendLog(fmt.Sprintf("%s ▶ %s", tag, query))

			err := downloadOne(idx+1, total, query)

			done := atomic.AddInt64(&completed, 1)
			pct := float64(done) / float64(total) * 100
			overallVal.Set(pct)

			if err != nil {
				appendLog(fmt.Sprintf("%s ❌ Falló: %v", tag, err))
			} else {
				atomic.AddInt64(&okCount, 1)
				appendLog(fmt.Sprintf("%s ✅ Completada.", tag))
			}
			statusText.Set(fmt.Sprintf("%d / %d", done, total))
		}(i, q)
	}

	wg.Wait()
	downloading = false
	overallVal.Set(100)

	ok := atomic.LoadInt64(&okCount)
	failed := int64(total) - ok
	if failed == 0 {
		appendLog(fmt.Sprintf("🎉 Todas las canciones descargadas (%d/%d) → %s", ok, total, outputDir))
		statusText.Set(fmt.Sprintf("✅ %d/%d completadas", ok, total))
	} else {
		appendLog(fmt.Sprintf("⚠  Finalizado: %d OK, %d con error.", ok, failed))
		statusText.Set(fmt.Sprintf("⚠ %d OK / %d errores", ok, failed))
	}
}

// ─── Single download ─────────────────────────────────────────────────────────

func downloadOne(idx, total int, query string) error {
	ytdlpPath := filepath.Join(appDir, "yt-dlp.exe")
	if runtime.GOOS != "windows" {
		ytdlpPath = filepath.Join(appDir, "yt-dlp")
	}

	tag := fmt.Sprintf("[%d/%d]", idx, total)
	arg := query
	if !strings.HasPrefix(query, "http://") && !strings.HasPrefix(query, "https://") {
		arg = "ytsearch1:" + query
		appendLog(fmt.Sprintf("   %s 🔎 Buscando: %s", tag, query))
	}

	outTemplate := filepath.Join(outputDir, "%(title)s.%(ext)s")
	args := []string{
		"--ffmpeg-location", appDir,
		"-x",
		"--audio-format", "mp3",
		"--audio-quality", "128K",
		"--newline",
		"--no-playlist",
		"-o", outTemplate,
		arg,
	}

	cmd := exec.Command(ytdlpPath, args...)
	cmd.Dir = appDir
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("iniciando yt-dlp: %w", err)
	}

	var lastPct float64
	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			if m := progressRegex.FindStringSubmatch(line); m != nil {
				pct, _ := strconv.ParseFloat(m[1], 64)
				// Log only every 25% jump to avoid spam
				if pct-lastPct >= 25 || pct >= 99 {
					appendLog(fmt.Sprintf("   %s %.0f%%", tag, pct))
					lastPct = pct
				}
			} else if strings.Contains(line, "[ExtractAudio]") {
				appendLog(fmt.Sprintf("   %s 🎵 Convirtiendo a MP3...", tag))
			} else if strings.Contains(line, "Destination:") {
				// Extract just the filename
				parts := strings.SplitN(line, "Destination: ", 2)
				if len(parts) == 2 {
					appendLog(fmt.Sprintf("   %s 💾 %s", tag, filepath.Base(parts[1])))
				}
			} else if strings.HasPrefix(line, "[youtube]") || strings.HasPrefix(line, "[ytsearch]") {
				appendLog(fmt.Sprintf("   %s %s", tag, line))
			}
		}
	}()

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "ERROR") {
				appendLog(fmt.Sprintf("   %s ⚠ %s", tag, line))
			}
		}
	}()

	return cmd.Wait()
}

// ─── Auto-download yt-dlp ────────────────────────────────────────────────────

func ensureYtDlp() error {
	name := "yt-dlp.exe"
	if runtime.GOOS != "windows" {
		name = "yt-dlp"
	}
	dest := filepath.Join(appDir, name)
	if fileExists(dest) {
		return nil
	}
	url := "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp.exe"
	if runtime.GOOS != "windows" {
		url = "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp"
	}
	appendLog("⬇  Descargando yt-dlp...")
	return downloadFile(url, dest, true)
}

// ─── Auto-download ffmpeg ────────────────────────────────────────────────────

func ensureFFmpeg() error {
	dest := filepath.Join(appDir, "ffmpeg.exe")
	if runtime.GOOS != "windows" {
		dest = filepath.Join(appDir, "ffmpeg")
	}
	if fileExists(dest) {
		return nil
	}
	if path, err := exec.LookPath("ffmpeg"); err == nil {
		appendLog("✅ ffmpeg encontrado en PATH: " + path)
		return nil
	}
	appendLog("⬇  Descargando ffmpeg (build esencial ~8MB)...")
	zipURL := "https://github.com/GyanD/codexffmpeg/releases/download/7.1/ffmpeg-7.1-essentials_build.zip"
	zipPath := filepath.Join(appDir, "ffmpeg_tmp.zip")
	if err := downloadFile(zipURL, zipPath, false); err != nil {
		return fmt.Errorf("descarga ffmpeg: %w", err)
	}
	defer os.Remove(zipPath)
	appendLog("📦 Extrayendo ffmpeg.exe...")
	return extractFFmpegFromZip(zipPath, dest)
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

func downloadFile(url, dest string, executable bool) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	if err != nil {
		return err
	}
	if executable {
		os.Chmod(dest, 0755)
	}
	return nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func shortPath(p string) string {
	home, _ := os.UserHomeDir()
	if strings.HasPrefix(p, home) {
		return "~" + p[len(home):]
	}
	return p
}

