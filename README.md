# 🎵 YouTube Downloader

Una aplicación portable y moderna en Go para descargar canciones desde YouTube, YouTube Music o buscar por nombre directamente a **128kbps en MP3**.

## ✨ Características

- **UI moderna** con tema oscuro (Fyne v2)
- **Descarga paralela** - hasta 3 canciones simultáneamente (configurable)
- **Cuadro de texto multilínea** - pega múltiples URLs/nombres, uno por línea
- **Barra de progreso** con porcentaje de canciones completadas
- **Log en tiempo real** con timestamps y etiquetas por canción
- **Selector de carpeta nativo** de Windows (cómodo y grande)
- **Auto-descarga de dependencias** - yt-dlp y ffmpeg se descargan automáticamente al primer inicio
- **Portable** - único ejecutable, sin instalación requerida
- **Cross-platform** - código compatible con Windows, Linux y macOS (aunque hay algunos detalles Windows-específicos)

## 🚀 Inicio rápido

### Requisitos
- **Windows 10/11** (probado en ambas versiones)
- **Go 1.21+** (solo si vas a compilar desde código)
- **GCC/MinGW** (para compilar, recomendado usar MSYS2)

### Usando el ejecutable precompilado

1. Descarga `YouTubeDownloader.exe` desde este repositorio
2. Ejecuta el `.exe`
   - **Primera vez:** Se descargarán automáticamente `yt-dlp.exe` (~10MB) y `ffmpeg.exe` (~8MB) en la misma carpeta
   - Las canciones se guardan en la carpeta `Musica/` junto al `.exe`
3. ¡A descargar!

### Compilar desde código

#### Requisitos previos

1. **Instalar Go**
   ```bash
   # Descarga desde https://golang.org/dl/
   # Verifica la instalación
   go version
   ```

2. **Instalar MSYS2 + GCC** (si no tienes compilador)
   ```bash
   # Descarga MSYS2 desde https://www.msys2.org/
   # Durante la instalación, selecciona la instalación con mingw64
   # Después de instalar, abre MSYS2 y ejecuta:
   pacman -S mingw-w64-ucrt-x86_64-gcc
   ```

3. **Clonar/descargar el repositorio**
   ```bash
   git clone https://github.com/tuusuario/YoutubeDownloader.git
   cd YoutubeDownloader
   ```

#### Compilación

```bash
# Opción 1: En Windows PowerShell (recomendado)
$env:PATH = "C:\msys64\ucrt64\bin;$env:PATH"
go build -ldflags="-H windowsgui -s -w" -o YouTubeDownloader.exe .

# Opción 2: Ejecutar el archivo build.bat (si está en el repositorio)
.\build.bat
```

**Explicación de flags:**
- `-H windowsgui` - Oculta la ventana de consola (solo muestra la UI de Fyne)
- `-s -w` - Minifica el ejecutable (sin debug info)

## 📖 Cómo usar

### Interfaz

```
┌─────────────────────────────────────┐
│   🎵 YouTube Downloader             │
├─────────────────────────────────────┤
│ [Cuadro de texto multilínea]        │
│  Pegá URLs o nombres aquí...        │
│                                     │
│ 📁 ~/Music/  [Cambiar carpeta]      │
├─────────────────────────────────────┤
│ [Log de descarga en tiempo real]    │
│ [15:04:05] ✅ yt-dlp listo.         │
│ [15:04:06] 📋 Iniciando 3 canciones │
│ [15:04:07] [1/3] ▶ Bohemian...      │
│ ...                                 │
├─────────────────────────────────────┤
│ Progreso: [===========     ] 2 / 3   │
└─────────────────────────────────────┘
```

### Ejemplos de entrada

Puedes mezclar URLs y nombres de canciones:

```
https://www.youtube.com/watch?v=dQw4w9WgXcQ
Bohemian Rhapsody Queen
https://music.youtube.com/watch?v=abcd1234
Hotel California Eagles
Shape of You Ed Sheeran
```

Presiona **"⬇ Descargar"** y las 5 canciones se descargarán en paralelo (máx 3 simultáneas).

### Cambiar carpeta de descarga

1. Click en **"Cambiar carpeta"**
2. Se abre el explorador nativo de Windows
3. Selecciona una carpeta y presiona OK
4. Las futuras descargas irán a esa carpeta

## 🏗️ Estructura del proyecto

```
YoutubeDownloader/
├── main.go                 # Código principal (UI + lógica de descarga)
├── zip_extract.go          # Extractor para ffmpeg.zip
├── go.mod                  # Módulos de Go
├── go.sum                  # Checksums de dependencias
├── build.bat               # Script de compilación (Windows)
├── YouTubeDownloader.exe   # Ejecutable compilado
├── README.md               # Este archivo
└── Musica/                 # Carpeta de descargas (creada automáticamente)
```

## 📦 Dependencias

### Externas (código)
- **fyne.io/fyne/v2** - Framework UI multiplataforma

### Externas (ejecutables)
Se descargan automáticamente:
- **yt-dlp.exe** - Descargador de YouTube (GitHub: yt-dlp/yt-dlp)
- **ffmpeg.exe** - Conversor de audio (GitHub: GyanD/codexffmpeg - build esencial)

### Librerías Go (automáticas)
Se instalan con `go mod tidy`:
- `golang.org/x/text` - Soporte de idiomas
- `golang.org/x/image` - Procesamiento de imágenes
- `github.com/go-gl/glfw/v3.3/glfw` - OpenGL
- Y muchas más... (se manejan automáticamente)

## ⚙️ Configuración avanzada

### Cambiar número máximo de descargas paralelas

Edita `main.go` línea ~24:

```go
const maxConcurrent = 3  // Cambia este número
```

**Recomendaciones:**
- `2-3` - Para conexiones normales
- `5+` - Si tienes fibra y YouTube no te bloquea
- `1` - Si tienes problemas de conexión

### Cambiar calidad de audio

Edita `main.go` en la función `downloadOne()`, busca:

```go
"--audio-quality", "128K",  // Cambia 128K por 192K, 256K, etc.
```

**Opciones:** `128K`, `192K`, `256K`, `320K` (máximo)

### Cambiar carpeta por defecto

Edita `main.go` línea ~36:

```go
outputDir = filepath.Join(appDir, "Musica")  // Cambia "Musica" por tu preferencia
```

## 🐛 Solución de problemas

### "❌ Error descargando yt-dlp"
- Verifica que tienes conexión a internet
- El firewall podría bloquear las descargas
- Intenta descargar manualmente desde: https://github.com/yt-dlp/yt-dlp/releases

### "⚠ Error con ffmpeg"
- Similar al anterior, verifica conexión
- O descarga manualmente desde: https://github.com/GyanD/codexffmpeg/releases
- Coloca `ffmpeg.exe` en la misma carpeta que el app

### "No encuentra yt-dlp"
- Asegúrate de que `yt-dlp.exe` está en la misma carpeta que `YouTubeDownloader.exe`
- Intenta ejecutar manualmente: `yt-dlp.exe --version`

### La descarga falla con "HTTP 429"
- YouTube te bloqueó temporalmente por demasiadas descargas
- Espera 1-2 horas o reduce `maxConcurrent` a 1
- Intenta usar VPN (yt-dlp soporta proxies)

### No descarga de YouTube Music
- Algunos videos de YT Music requieren autenticación
- Prueba con una búsqueda de nombre en su lugar: `"Nombre Canción Artista"`

## 💡 Tips de uso

1. **Playlist de Spotify → YouTube** - Usa un converter online para obtener URLs de YouTube, luego cópialas al app
2. **Busca por nombre** - `"All Too Well Taylor Swift"` funciona mejor que URLs crípticas
3. **Combina formatos** - Puedes mezclar URLs y nombres sin problema
4. **Renombra después** - Los MP3s se guardan con el título de YouTube, puedes renombrarlos después
5. **Deja el .exe siempre con ffmpeg y yt-dlp** - No muevas el app a otra carpeta sin las dependencias

## 🔒 Privacidad

- Este app es **offline** (excepto al descargar yt-dlp/ffmpeg la primera vez)
- Solo se conecta a:
  - GitHub (para descargar yt-dlp/ffmpeg)
  - YouTube (para descargar videos)
- No se envía telemetría ni tracking

## 📄 Licencia

MIT - Siéntete libre de usar, modificar y distribuir

## 👨‍💻 Desarrollo

### Compilación con debug

```bash
go build -o YouTubeDownloader.exe .  # Sin -ldflags para debug
```

### Cross-compile a Linux

```bash
GOOS=linux GOARCH=amd64 go build -o YouTubeDownloader-linux .
```

### Ver logs en terminal

Para debug, edita `main.go` y remueve `-H windowsgui` de los ldflags antes de compilar.

## 🤝 Contribuciones

Pull requests bienvenidos. Para cambios grandes, abre una issue primero.

## 📝 Changelog

### v1.0 - Inicial
- ✅ Descarga de canciones desde YouTube a 128kbps MP3
- ✅ Descarga paralela (máx 3)
- ✅ UI moderna con Fyne
- ✅ Auto-descarga de dependencias
- ✅ Selector nativo de carpeta Windows

---

**Made with ❤️ in Go**

¿Preguntas? Abre un issue en GitHub o contacta directamente.

