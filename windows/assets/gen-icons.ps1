# gen-icons.ps1 - regenerates Hotfix's Windows icon assets from the fire emoji
# (U+1F525), so the tray icon and the .exe/installer icon match the flame shown
# next to "Hotfix Settings" on the settings page.
#
# Produces:
#   - flame16.png / flame32.png : tray icons (the color fire emoji)
#   - Hotfix.ico                : multi-size colored app icon (exe + installer)
#
# The glyph is rendered as the exact Segoe UI Emoji color flame by taking a
# transparent-background screenshot through headless Edge (GDI+ only renders
# emoji monochrome). Edge ships with Windows, so no extra dependency.
#
# This file is intentionally ASCII-only: Windows PowerShell 5.1 reads .ps1 files
# as ANSI when there's no BOM, so the emoji is injected via an HTML entity below.
#
# Run on Windows:  powershell -ExecutionPolicy Bypass -File gen-icons.ps1
# Commit the regenerated PNG/ICO files; CI consumes them as-is.

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.Drawing

$here = Split-Path -Parent $MyInvocation.MyCommand.Path

# --- 1. Render the emoji at 256px with a transparent background via Edge ---

$edge = @(
    "${env:ProgramFiles(x86)}\Microsoft\Edge\Application\msedge.exe",
    "$env:ProgramFiles\Microsoft\Edge\Application\msedge.exe"
) | Where-Object { Test-Path $_ } | Select-Object -First 1
if (-not $edge) { throw "msedge.exe not found - needed to render the color emoji." }

$tmpHtml = Join-Path $env:TEMP "hotfix-emoji.html"
$tmpPng = Join-Path $env:TEMP "hotfix-emoji.png"
Remove-Item -Force $tmpPng -ErrorAction SilentlyContinue
@'
<!doctype html><html><head><meta charset="utf-8"></head>
<body style="margin:0;padding:0;width:256px;height:256px;display:flex;align-items:center;justify-content:center">
<div style="font-size:232px;line-height:1;font-family:'Segoe UI Emoji'">&#x1F525;</div>
</body></html>
'@ | Out-File -FilePath $tmpHtml -Encoding utf8

$fileUrl = "file:///" + ($tmpHtml -replace '\\', '/')
& $edge --headless=new --disable-gpu --hide-scrollbars --force-device-scale-factor=1 `
    --default-background-color=00000000 --window-size=256,256 `
    --screenshot="$tmpPng" $fileUrl 2>$null

for ($i = 0; $i -lt 40 -and -not (Test-Path $tmpPng); $i++) { Start-Sleep -Milliseconds 250 }
if (-not (Test-Path $tmpPng)) { throw "Edge did not produce the emoji screenshot." }
$src = [System.Drawing.Image]::FromFile($tmpPng)
Write-Host "  rendered emoji: $($src.Width)x$($src.Height)"

# --- 2. High-quality downscale to each target size ---

function Resize-To([System.Drawing.Image]$img, [int]$size) {
    $bmp = New-Object System.Drawing.Bitmap($size, $size, [System.Drawing.Imaging.PixelFormat]::Format32bppArgb)
    $g = [System.Drawing.Graphics]::FromImage($bmp)
    $g.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
    $g.PixelOffsetMode = [System.Drawing.Drawing2D.PixelOffsetMode]::HighQuality
    $g.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::AntiAlias
    $g.Clear([System.Drawing.Color]::Transparent)
    $g.DrawImage($img, 0, 0, $size, $size)
    $g.Dispose()
    return $bmp
}

# Encode a Bitmap as a 32bpp BMP/DIB icon frame: a BITMAPINFOHEADER with doubled
# height, a bottom-up BGRA pixel array, then an all-zero AND mask (transparency
# comes from the alpha channel). Broadest-compatibility ICO frame format.
function Get-DibBytes([System.Drawing.Bitmap]$bmp) {
    $w = $bmp.Width; $h = $bmp.Height
    $ms = New-Object System.IO.MemoryStream
    $bw = New-Object System.IO.BinaryWriter($ms)
    $bw.Write([uint32]40); $bw.Write([int32]$w); $bw.Write([int32]($h * 2))
    $bw.Write([uint16]1); $bw.Write([uint16]32); $bw.Write([uint32]0); $bw.Write([uint32]0)
    $bw.Write([int32]0); $bw.Write([int32]0); $bw.Write([uint32]0); $bw.Write([uint32]0)
    for ($y = $h - 1; $y -ge 0; $y--) {
        for ($x = 0; $x -lt $w; $x++) {
            $p = $bmp.GetPixel($x, $y)
            $bw.Write([byte]$p.B); $bw.Write([byte]$p.G); $bw.Write([byte]$p.R); $bw.Write([byte]$p.A)
        }
    }
    $maskRow = [Math]::Floor(($w + 31) / 32) * 4
    $bw.Write((New-Object byte[] ($maskRow * $h)))
    $bw.Flush()
    $bytes = $ms.ToArray()
    $bw.Dispose(); $ms.Dispose()
    return , $bytes
}

# Assemble a multi-frame ICO from DIB frames (mirrors pngToICO in main.go).
function Save-Ico([hashtable[]]$frames, [string]$outFile) {
    $ms = New-Object System.IO.MemoryStream
    $bw = New-Object System.IO.BinaryWriter($ms)
    $bw.Write([uint16]0); $bw.Write([uint16]1); $bw.Write([uint16]$frames.Count)
    $offset = 6 + 16 * $frames.Count
    foreach ($f in $frames) {
        $wb = if ([int]$f.size -ge 256) { 0 } else { [int]$f.size }
        $bw.Write([byte]$wb); $bw.Write([byte]$wb); $bw.Write([byte]0); $bw.Write([byte]0)
        $bw.Write([uint16]1); $bw.Write([uint16]32)
        $bw.Write([uint32]$f.bytes.Length); $bw.Write([uint32]$offset)
        $offset += $f.bytes.Length
    }
    foreach ($f in $frames) { $bw.Write($f.bytes) }
    $bw.Flush()
    [System.IO.File]::WriteAllBytes($outFile, $ms.ToArray())
    $bw.Dispose(); $ms.Dispose()
    Write-Host "  wrote $outFile ($($frames.Count) frames)"
}

Write-Host "Generating tray PNGs..."
foreach ($sz in 16, 32) {
    $b = Resize-To $src $sz
    $b.Save((Join-Path $here "flame$sz.png"), [System.Drawing.Imaging.ImageFormat]::Png)
    $b.Dispose()
    Write-Host "  wrote flame$sz.png"
}

Write-Host "Generating Hotfix.ico..."
$frames = @()
foreach ($sz in 16, 24, 32, 48, 64, 128, 256) {
    $b = Resize-To $src $sz
    $frames += @{ size = $sz; bytes = (Get-DibBytes $b) }
    $b.Dispose()
}
Save-Ico $frames (Join-Path $here "Hotfix.ico")

$src.Dispose()
Write-Host "Done."
