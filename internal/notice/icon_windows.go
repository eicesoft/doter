//go:build windows

package notice

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

// iconCache is bounded by the number of distinct apps that notify.
var iconCache = struct {
	sync.Mutex
	icons map[string]string
}{icons: make(map[string]string)}

// AppIcon returns a PNG data URL for the app's icon, or "" when it cannot be
// resolved.
func AppIcon(aumid string) string {
	if aumid == "" {
		return ""
	}

	iconCache.Lock()
	data, ok := iconCache.icons[aumid]
	iconCache.Unlock()
	if ok {
		return data
	}

	data = loadAppIcon(aumid)
	iconCache.Lock()
	iconCache.icons[aumid] = data
	iconCache.Unlock()
	return data
}

// iconScript resolves an icon in Windows PowerShell. The AUMID arrives through
// the environment, so it never becomes part of the script text. Packaged apps
// expose their logo in AppxManifest.xml; plain Win32 ids are looked up as an
// executable.
const iconScript = `$ErrorActionPreference = 'SilentlyContinue'
$id = $env:DOTER_AUMID
$path = $null

if ($id.Contains('!')) {
    $family = $id.Split('!')[0]
    $cut = $family.LastIndexOf('_')
    if ($cut -gt 0) { $name = $family.Substring(0, $cut) } else { $name = $family }
    $pkg = Get-AppxPackage -Name $name | Where-Object { $_.PackageFamilyName -eq $family } | Select-Object -First 1
    if ($pkg) {
        $manifest = [xml](Get-Content -LiteralPath (Join-Path $pkg.InstallLocation 'AppxManifest.xml') -Raw)
        $logo = $manifest.Package.Properties.Logo
        if ($logo) {
            $candidate = Join-Path $pkg.InstallLocation $logo
            $dir = Split-Path $candidate
            $stem = [IO.Path]::GetFileNameWithoutExtension($candidate)
            $path = (Get-ChildItem -LiteralPath $dir -Filter "$stem*.png" | Sort-Object Length | Select-Object -First 1).FullName
        }
    }
}

Add-Type -AssemblyName System.Drawing
$img = $null
if ($path) { $img = [Drawing.Image]::FromFile($path) }
if (-not $img) {
    $exe = $id
    if (-not (Test-Path -LiteralPath $exe)) {
        $exe = (Get-Command $id).Source
        if (-not $exe -and $id -notlike '*\*') { $exe = (Get-Command ($id + '.exe')).Source }
    }
    if ($exe) {
        $icon = [Drawing.Icon]::ExtractAssociatedIcon($exe)
        if ($icon) { $img = $icon.ToBitmap() }
    }
}
if (-not $img) { exit 1 }

$size = 64
$bmp = New-Object Drawing.Bitmap $size, $size
$g = [Drawing.Graphics]::FromImage($bmp)
$g.InterpolationMode = [Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
$g.Clear([Drawing.Color]::Transparent)
$ratio = [Math]::Min($size / $img.Width, $size / $img.Height)
$w = [int]($img.Width * $ratio)
$h = [int]($img.Height * $ratio)
$g.DrawImage($img, [int](($size - $w) / 2), [int](($size - $h) / 2), $w, $h)
$stream = New-Object IO.MemoryStream
$bmp.Save($stream, [Drawing.Imaging.ImageFormat]::Png)
Write-Output ([Convert]::ToBase64String($stream.ToArray()))
`

func loadAppIcon(aumid string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "powershell", "-NoProfile", "-NonInteractive", "-Command", iconScript)
	cmd.Env = append(os.Environ(), "DOTER_AUMID="+aumid)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	data := strings.TrimSpace(string(out))
	if data == "" {
		return ""
	}
	return "data:image/png;base64," + data
}
