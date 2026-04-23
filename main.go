package main

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/binding"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

const defaultConcurrency = 3
const highConcurrencyWarningThreshold = 8

const prefOutputDir = "output_dir"
const prefAudioQuality = "audio_quality"
const prefDownloadMethod = "download_method"
const prefConcurrency = "parallel_concurrency"

var allowedAudioQualities = []string{"64K", "96K", "128K", "160K", "192K", "256K", "320K"}
var allowedDownloadMethods = []string{"Auto", "Normal", "Alternativo"}
var allowedConcurrencyOptions = []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "12"}

const defaultAudioQuality = "128K"
const defaultDownloadMethod = "Auto"

type failedItem struct {
	name  string
	query string
}

type iconHintButton struct {
	*widget.Button
	hint    string
	onHover func(string)
}

func newIconHintButton(icon fyne.Resource, hint string, onHover func(string), tapped func()) *iconHintButton {
	btn := widget.NewButtonWithIcon("", icon, tapped)
	btn.Importance = widget.LowImportance
	return &iconHintButton{Button: btn, hint: hint, onHover: onHover}
}

func newTextHintButton(label string, hint string, onHover func(string), tapped func()) *iconHintButton {
	btn := widget.NewButton(label, tapped)
	btn.Importance = widget.LowImportance
	btn.Alignment = widget.ButtonAlignLeading
	return &iconHintButton{Button: btn, hint: hint, onHover: onHover}
}

func (b *iconHintButton) MouseIn(ev *desktop.MouseEvent) {
	b.Button.MouseIn(ev)
	if b.onHover != nil {
		b.onHover(b.hint)
	}
}

func (b *iconHintButton) MouseOut() {
	b.Button.MouseOut()
	if b.onHover != nil {
		b.onHover("")
	}
}

func (b *iconHintButton) MouseMoved(ev *desktop.MouseEvent) {
	b.Button.MouseMoved(ev)
}

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
	prefs := a.Preferences()

	savedOutputDir := strings.TrimSpace(prefs.String(prefOutputDir))
	if savedOutputDir != "" {
		outputDir = savedOutputDir
		_ = os.MkdirAll(outputDir, 0755)
	}

	w := a.NewWindow("🎵 YouTube Downloader")
	w.Resize(fyne.NewSize(860, 700))
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

	hintText := binding.NewString()
	hintText.Set("")
	hintLabel := widget.NewLabelWithData(hintText)
	outputLabel := widget.NewLabel(shortPath(outputDir))

	setHoverHint := func(msg string) {
		if strings.TrimSpace(msg) == "" {
			hintText.Set("")
			return
		}
		hintText.Set(msg)
	}

	outputBtn := newIconHintButton(theme.FolderOpenIcon(), "Cambiar carpeta de destino", setHoverHint, func() {
		go func() {
			path, err := pickFolderWindows()
			if err != nil {
				appendLog("⚠  No se pudo abrir el selector de carpeta: " + err.Error())
				return
			}
			if path == "" {
				return
			}
			outputDir = path
			prefs.SetString(prefOutputDir, path)
			outputLabel.SetText(shortPath(path))
		}()
	})
	clearLogsBtn := newIconHintButton(theme.DeleteIcon(), "Limpiar panel de logs", setHoverHint, func() {
		logText.Set("")
		appendLog("🧹 Logs limpiados.")
	})

	qualityLabel := widget.NewLabel("Calidad")
	qualitySelect := widget.NewSelect(allowedAudioQualities, func(value string) {
		prefs.SetString(prefAudioQuality, normalizeAudioQuality(value))
	})
	qualitySelect.SetSelected(normalizeAudioQuality(prefs.StringWithFallback(prefAudioQuality, defaultAudioQuality)))
	concurrencyLabel := widget.NewLabel("Paralelo")
	concurrencySelect := widget.NewSelect(allowedConcurrencyOptions, func(value string) {
		prefs.SetInt(prefConcurrency, normalizeConcurrency(value))
	})
	concurrencySelect.SetSelected(strconv.Itoa(normalizeConcurrency(strconv.Itoa(prefs.IntWithFallback(prefConcurrency, defaultConcurrency)))))
	methodLabel := widget.NewLabel("Metodo")
	methodSelect := widget.NewSelect(allowedDownloadMethods, func(value string) {
		prefs.SetString(prefDownloadMethod, normalizeDownloadMethod(value))
	})
	methodSelect.SetSelected(normalizeDownloadMethod(prefs.StringWithFallback(prefDownloadMethod, defaultDownloadMethod)))

	var downloadBtn *iconHintButton
	downloadBtn = newIconHintButton(theme.DownloadIcon(), "Iniciar descargas", setHoverHint, func() {
		raw := strings.TrimSpace(urlEntry.Text)
		if raw == "" {
			appendLog("⚠  Ingresá al menos una URL o nombre de canción.")
			return
		}
		if downloading {
			return
		}
		queries, duplicates := normalizeQueries(raw)
		if len(queries) == 0 {
			appendLog("⚠  No hay entradas validas para descargar.")
			return
		}
		if duplicates > 0 {
			appendLog(fmt.Sprintf("ℹ  Se omitieron %d entrada(s) duplicada(s).", duplicates))
		}
		selectedQuality := normalizeAudioQuality(qualitySelect.Selected)
		selectedConcurrency := normalizeConcurrency(concurrencySelect.Selected)
		selectedMethod := normalizeDownloadMethod(methodSelect.Selected)
		if selectedConcurrency >= highConcurrencyWarningThreshold {
			appendLog(fmt.Sprintf("⚠  Paralelismo alto (%d). Si ves fallos o lentitud, proba con 4-6.", selectedConcurrency))
		}
		downloadBtn.Disable()
		go func() {
			startBatchDownload(queries, selectedQuality, selectedMethod, selectedConcurrency)
			downloadBtn.Enable()
		}()
	})
	downloadBtn.Importance = widget.HighImportance

	progressRow := container.NewBorder(nil, nil, widget.NewLabel("Progreso"), statusLabel, overallBar)

	settingsGrid := container.NewGridWithColumns(3,
		container.NewVBox(qualityLabel, qualitySelect),
		container.NewVBox(concurrencyLabel, concurrencySelect),
		container.NewVBox(methodLabel, methodSelect),
	)

	actionRow := container.NewHBox(
		clearLogsBtn,
		outputBtn,
		outputLabel,
		layout.NewSpacer(),
		downloadBtn,
	)

	inputCard := widget.NewCard("Entradas", "Pegá URLs o nombres (una por linea)", entryScroll)
	controlsCard := widget.NewCard("Controles", "Configuracion de descarga", container.NewVBox(
		actionRow,
		hintLabel,
		widget.NewSeparator(),
		settingsGrid,
	))
	logsCard := widget.NewCard("Actividad", "Eventos importantes", logScroll)
	progressCard := widget.NewCard("Estado", "", progressRow)

	content := container.NewVBox(
		inputCard,
		controlsCard,
		logsCard,
		progressCard,
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
		appendLog(fmt.Sprintf("🎵 Listo. Elegi calidad, metodo y paralelo (recomendado 4-6). Default: %d.", defaultConcurrency))
	}()

	w.ShowAndRun()
}

// ─── Windows native folder picker ───────────────────────────────────────────

func pickFolderWindows() (string, error) {
	cmd := exec.Command("powershell.exe", "-NoProfile", "-STA", "-Command",
		`Add-Type -AssemblyName System.Windows.Forms;`+
			`$d = New-Object System.Windows.Forms.FolderBrowserDialog;`+
			`$d.Description = 'Seleccionar carpeta de destino';`+
			`$d.ShowNewFolderButton = $true;`+
			`if ($d.ShowDialog() -eq 'OK') { Write-Output $d.SelectedPath }`)
	hideConsole(cmd)
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

func startBatchDownload(queries []string, quality string, method string, concurrency int) {
	downloading = true
	total := len(queries)
	quality = normalizeAudioQuality(quality)
	method = normalizeDownloadMethod(method)
	concurrency = normalizeConcurrency(strconv.Itoa(concurrency))
	overallVal.Set(0)
	statusText.Set(fmt.Sprintf("0 / %d", total))
	appendLog(fmt.Sprintf("📋 Iniciando %d descarga(s) (paralelo %d, %s, metodo %s).", total, concurrency, quality, method))

	var completed int64
	var okCount int64
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	var failedMu sync.Mutex
	failedItems := make([]failedItem, 0)

	for i, q := range queries {
		wg.Add(1)
		go func(idx int, query string) {
			defer wg.Done()
			sem <- struct{}{}        // acquire
			defer func() { <-sem }() // release

			tag := fmt.Sprintf("[%d/%d]", idx+1, total)
			appendLog(fmt.Sprintf("%s ▶ Iniciando", tag))

			videoName, methodUsed, err := downloadOne(query, quality, method)
			displayName := pickDisplayName(videoName, query)

			done := atomic.AddInt64(&completed, 1)
			pct := float64(done) / float64(total) * 100
			overallVal.Set(pct)

			if err != nil {
				appendLog(fmt.Sprintf("%s ❌ %s", tag, displayName))
				failedMu.Lock()
				failedItems = append(failedItems, failedItem{name: displayName, query: query})
				failedMu.Unlock()
			} else {
				atomic.AddInt64(&okCount, 1)
				appendLog(fmt.Sprintf("%s ✅ %s (%s)", tag, displayName, methodUsed))
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
		failedMu.Lock()
		appendLog("📌 Fallaron estas descargas:")
		for _, item := range failedItems {
			appendLog(fmt.Sprintf("   • %s", item.name))
		}
		failedMu.Unlock()
		statusText.Set(fmt.Sprintf("⚠ %d OK / %d errores", ok, failed))
	}
}

// ─── Single download ─────────────────────────────────────────────────────────

func downloadOne(query string, quality string, method string) (string, string, error) {
	ytdlpPath := filepath.Join(appDir, "yt-dlp.exe")
	if runtime.GOOS != "windows" {
		ytdlpPath = filepath.Join(appDir, "yt-dlp")
	}
	quality = normalizeAudioQuality(quality)
	method = normalizeDownloadMethod(method)

	arg := query
	if !strings.HasPrefix(query, "http://") && !strings.HasPrefix(query, "https://") {
		arg = "ytsearch1:" + query
	}

	attempts := methodsForSelection(method)
	var lastErr error
	bestTitle := ""
	bestMethod := attempts[0]
	for i, attempt := range attempts {
		if i > 0 {
			appendLog(fmt.Sprintf("↻ Reintentando con metodo %s...", attempt))
		}
		title, err := runYtDlp(ytdlpPath, arg, quality, attempt)
		if title != "" {
			bestTitle = title
		}
		if err == nil {
			return bestTitle, attempt, nil
		}
		lastErr = err
		bestMethod = attempt
	}

	if bestTitle == "" {
		bestTitle = query
	}
	return bestTitle, bestMethod, lastErr
}

func runYtDlp(ytdlpPath, arg, quality, method string) (string, error) {
	outTemplate := filepath.Join(outputDir, "%(title)s.%(ext)s")
	args := []string{
		"--ffmpeg-location", appDir,
		"-x",
		"--audio-format", "mp3",
		"--audio-quality", quality,
		"--no-playlist",
		"--print", "before_dl:%(title)s",
		"-o", outTemplate,
	}
	if method == "Alternativo" {
		args = append(args,
			"--extractor-args", "youtube:player_client=android,web",
			"--force-ipv4",
			"--extractor-retries", "5",
		)
	}
	args = append(args, arg)

	cmd := exec.Command(ytdlpPath, args...)
	cmd.Dir = appDir
	hideConsole(cmd)

	stdout, _ := cmd.StdoutPipe()
	stderr, _ := cmd.StderrPipe()

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("iniciando yt-dlp: %w", err)
	}

	var wg sync.WaitGroup
	var mu sync.Mutex
	videoName := ""
	firstErrLine := ""

	wg.Add(2)
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.HasPrefix(line, "before_dl:") {
				mu.Lock()
				videoName = strings.TrimSpace(strings.TrimPrefix(line, "before_dl:"))
				mu.Unlock()
				continue
			}
			if strings.Contains(line, "Destination: ") {
				parts := strings.SplitN(line, "Destination: ", 2)
				if len(parts) == 2 {
					base := filepath.Base(strings.TrimSpace(parts[1]))
					mu.Lock()
					if videoName == "" {
						videoName = strings.TrimSuffix(base, filepath.Ext(base))
					}
					mu.Unlock()
				}
			}
		}
	}()

	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(strings.ToUpper(line), "ERROR") {
				mu.Lock()
				if firstErrLine == "" {
					firstErrLine = line
				}
				mu.Unlock()
			}
		}
	}()

	err := cmd.Wait()
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	if err != nil {
		if firstErrLine != "" {
			return videoName, fmt.Errorf("%s", firstErrLine)
		}
		return videoName, err
	}
	return videoName, nil
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

func normalizeQueries(raw string) ([]string, int) {
	lines := strings.Split(raw, "\n")
	seen := make(map[string]struct{}, len(lines))
	queries := make([]string, 0, len(lines))
	duplicates := 0
	for _, line := range lines {
		q := strings.TrimSpace(line)
		if q == "" {
			continue
		}
		key := strings.ToLower(q)
		if _, exists := seen[key]; exists {
			duplicates++
			continue
		}
		seen[key] = struct{}{}
		queries = append(queries, q)
	}
	return queries, duplicates
}

func openFolder(path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("ruta vacia")
	}
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("creando carpeta: %w", err)
	}
	cleanPath := filepath.Clean(path)
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		absPath = cleanPath
	}
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "windows":
		cmd = exec.Command("explorer.exe", "/select,", absPath)
	case "darwin":
		cmd = exec.Command("open", absPath)
	default:
		cmd = exec.Command("xdg-open", absPath)
	}
	hideConsole(cmd)
	if err := cmd.Start(); err == nil {
		return nil
	}
	if runtime.GOOS == "windows" {
		fallback := exec.Command("cmd", "/C", "start", "", "\""+absPath+"\"")
		hideConsole(fallback)
		return fallback.Start()
	}
	return cmd.Start()
}

func normalizeAudioQuality(quality string) string {
	for _, v := range allowedAudioQualities {
		if quality == v {
			return quality
		}
	}
	return defaultAudioQuality
}

func normalizeConcurrency(raw string) int {
	v, err := strconv.Atoi(strings.TrimSpace(raw))
	if err != nil || v < 1 {
		return defaultConcurrency
	}
	for _, allowed := range allowedConcurrencyOptions {
		if allowed == strconv.Itoa(v) {
			return v
		}
	}
	return defaultConcurrency
}

func normalizeDownloadMethod(method string) string {
	for _, v := range allowedDownloadMethods {
		if method == v {
			return method
		}
	}
	return defaultDownloadMethod
}

func methodsForSelection(method string) []string {
	method = normalizeDownloadMethod(method)
	if method == "Auto" {
		return []string{"Normal", "Alternativo"}
	}
	return []string{method}
}

func pickDisplayName(videoName, query string) string {
	if strings.TrimSpace(videoName) != "" {
		return videoName
	}
	return query
}
