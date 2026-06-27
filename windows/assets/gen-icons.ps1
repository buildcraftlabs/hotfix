# gen-icons.ps1 — regenerates Hotfix's Windows icon assets.
#
# Produces, from the flame outline in icon/AppIcon.svg:
#   • tray_white16.png / tray_white32.png — monochrome white flame (dark taskbar)
#   • tray_black16.png / tray_black32.png — monochrome black flame (light taskbar)
#   • Hotfix.ico — multi-size colored app icon (orange flame on dark rounded rect)
#     embedded into the .exe (goversioninfo) and used as the installer icon.
#
# The tray PNGs mirror the macOS menu-bar flame, which is a monochrome SF Symbol
# that adapts to the menu bar's appearance; main.go picks white vs black at
# runtime from the SystemUsesLightTheme registry value.
#
# Run on Windows with .NET available:  powershell -ExecutionPolicy Bypass -File gen-icons.ps1
# Commit the regenerated PNG/ICO files; CI consumes them as-is.

Set-StrictMode -Version Latest
$ErrorActionPreference = 'Stop'
Add-Type -AssemblyName System.Drawing

$here = Split-Path -Parent $MyInvocation.MyCommand.Path

# Flame outline from icon/AppIcon.svg ("Primary flame body"), in the 512x512
# viewBox. Start point followed by cubic-bezier segments (c1x,c1y, c2x,c2y, x,y).
$start = @(256.0, 88.0)
$segs = @(
    @(256, 88, 308, 136, 318, 184),
    @(326, 222, 300, 238, 308, 268),
    @(316, 296, 344, 288, 350, 310),
    @(364, 356, 336, 406, 292, 426),
    @(270, 436, 242, 440, 222, 436),
    @(180, 428, 150, 402, 144, 368),
    @(136, 330, 164, 306, 168, 272),
    @(172, 244, 152, 214, 158, 184),
    @(164, 152, 182, 124, 202, 106),
    @(218, 91, 238, 82, 256, 88)
)
# Bounding box of the control points above (good enough for centring).
$minX = 136.0; $maxX = 364.0; $minY = 82.0; $maxY = 440.0
$flameW = $maxX - $minX
$flameH = $maxY - $minY

# Build a GraphicsPath for the flame, scaled by $s and offset by ($ox,$oy).
function New-FlamePath([double]$s, [double]$ox, [double]$oy) {
    $p = New-Object System.Drawing.Drawing2D.GraphicsPath
    $cx = $start[0] * $s + $ox
    $cy = $start[1] * $s + $oy
    foreach ($g in $segs) {
        $p.AddBezier(
            [single]$cx, [single]$cy,
            [single]($g[0] * $s + $ox), [single]($g[1] * $s + $oy),
            [single]($g[2] * $s + $ox), [single]($g[3] * $s + $oy),
            [single]($g[4] * $s + $ox), [single]($g[5] * $s + $oy))
        $cx = $g[4] * $s + $ox
        $cy = $g[5] * $s + $oy
    }
    $p.CloseFigure()
    return $p
}

# Fit transform: scale the flame into a $size canvas leaving $pad fraction margin.
function Get-Fit([int]$size, [double]$pad) {
    $avail = $size * (1.0 - 2.0 * $pad)
    $scale = [Math]::Min($avail / $flameW, $avail / $flameH)
    $ox = ($size - $flameW * $scale) / 2.0 - $minX * $scale
    $oy = ($size - $flameH * $scale) / 2.0 - $minY * $scale
    return @{ scale = $scale; ox = $ox; oy = $oy }
}

function New-Canvas([int]$size) {
    $bmp = New-Object System.Drawing.Bitmap($size, $size, [System.Drawing.Imaging.PixelFormat]::Format32bppArgb)
    $g = [System.Drawing.Graphics]::FromImage($bmp)
    $g.SmoothingMode = [System.Drawing.Drawing2D.SmoothingMode]::AntiAlias
    $g.InterpolationMode = [System.Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
    $g.PixelOffsetMode = [System.Drawing.Drawing2D.PixelOffsetMode]::HighQuality
    $g.Clear([System.Drawing.Color]::Transparent)
    return @{ bmp = $bmp; g = $g }
}

# Monochrome flame (white or black) on transparent background → PNG file.
function Save-MonoPng([int]$size, [System.Drawing.Color]$color, [string]$outFile) {
    $c = New-Canvas $size
    $fit = Get-Fit $size 0.08
    $path = New-FlamePath $fit.scale $fit.ox $fit.oy
    $brush = New-Object System.Drawing.SolidBrush($color)
    $c.g.FillPath($brush, $path)
    $brush.Dispose(); $path.Dispose(); $c.g.Dispose()
    $c.bmp.Save($outFile, [System.Drawing.Imaging.ImageFormat]::Png)
    $c.bmp.Dispose()
    Write-Host "  wrote $outFile"
}

# Colored app icon → Bitmap (orange flame on dark rounded rect, with ">_").
function Render-ColorBitmap([int]$size) {
    $c = New-Canvas $size
    $g = $c.g

    # Dark rounded-rect background (#141416).
    $radius = [single]($size * 0.22)
    $d = [single]($radius * 2.0)
    $bg = New-Object System.Drawing.Drawing2D.GraphicsPath
    $bg.AddArc(0, 0, $d, $d, 180, 90)
    $bg.AddArc([single]($size - $d), 0, $d, $d, 270, 90)
    $bg.AddArc([single]($size - $d), [single]($size - $d), $d, $d, 0, 90)
    $bg.AddArc(0, [single]($size - $d), $d, $d, 90, 90)
    $bg.CloseFigure()
    $bgBrush = New-Object System.Drawing.SolidBrush([System.Drawing.Color]::FromArgb(255, 20, 20, 22))
    $g.FillPath($bgBrush, $bg)

    # Flame filled with a vertical gradient (#C9461E bottom → #FF8C42 top).
    $fit = Get-Fit $size 0.20
    $path = New-FlamePath $fit.scale $fit.ox $fit.oy
    $rect = New-Object System.Drawing.RectangleF(0, [single]($minY * $fit.scale + $fit.oy), [single]$size, [single]($flameH * $fit.scale))
    $grad = New-Object System.Drawing.Drawing2D.LinearGradientBrush(
        $rect,
        [System.Drawing.Color]::FromArgb(255, 255, 140, 66),
        [System.Drawing.Color]::FromArgb(255, 201, 70, 30),
        [System.Drawing.Drawing2D.LinearGradientMode]::Vertical)
    $g.FillPath($grad, $path)

    # ">_" prompt, white, centred low in the flame (scaled to icon size).
    if ($size -ge 32) {
        $fontSize = [single]($size * 0.17)
        $font = New-Object System.Drawing.Font("Consolas", $fontSize, [System.Drawing.FontStyle]::Bold, [System.Drawing.GraphicsUnit]::Pixel)
        $fmt = New-Object System.Drawing.StringFormat
        $fmt.Alignment = [System.Drawing.StringAlignment]::Center
        $fmt.LineAlignment = [System.Drawing.StringAlignment]::Center
        $white = New-Object System.Drawing.SolidBrush([System.Drawing.Color]::FromArgb(242, 255, 255, 255))
        $textRect = New-Object System.Drawing.RectangleF(0, [single]($size * 0.52), [single]$size, [single]($size * 0.22))
        $g.DrawString(">_", $font, $white, $textRect, $fmt)
        $font.Dispose(); $white.Dispose()
    }

    $g.Dispose()
    return $c.bmp
}

# Encode a Bitmap as a 32bpp BMP/DIB icon frame: a BITMAPINFOHEADER with
# doubled height, a bottom-up BGRA pixel array, then an all-zero AND mask
# (transparency comes from the alpha channel). This is the most widely
# compatible ICO frame format for sizes up to 128px.
function Get-DibBytes([System.Drawing.Bitmap]$bmp) {
    $w = $bmp.Width; $h = $bmp.Height
    $ms = New-Object System.IO.MemoryStream
    $bw = New-Object System.IO.BinaryWriter($ms)

    # BITMAPINFOHEADER (40 bytes).
    $bw.Write([uint32]40)        # biSize
    $bw.Write([int32]$w)         # biWidth
    $bw.Write([int32]($h * 2))   # biHeight (color + AND mask)
    $bw.Write([uint16]1)         # biPlanes
    $bw.Write([uint16]32)        # biBitCount
    $bw.Write([uint32]0)         # biCompression = BI_RGB
    $bw.Write([uint32]0)         # biSizeImage
    $bw.Write([int32]0); $bw.Write([int32]0)   # resolution
    $bw.Write([uint32]0); $bw.Write([uint32]0) # palette

    # Color array, bottom-up, BGRA per pixel.
    for ($y = $h - 1; $y -ge 0; $y--) {
        for ($x = 0; $x -lt $w; $x++) {
            $p = $bmp.GetPixel($x, $y)
            $bw.Write([byte]$p.B); $bw.Write([byte]$p.G); $bw.Write([byte]$p.R); $bw.Write([byte]$p.A)
        }
    }

    # AND mask: 1bpp, rows padded to 4-byte boundary, all zero = fully opaque.
    $maskRow = [Math]::Floor(($w + 31) / 32) * 4
    $zero = New-Object byte[] ($maskRow * $h)
    $bw.Write($zero)

    $bw.Flush()
    $bytes = $ms.ToArray()
    $bw.Dispose(); $ms.Dispose()
    return ,$bytes
}

# Assemble a multi-frame ICO from PNG frames (Vista+ supports PNG-compressed
# frames; required for the 256px frame). Mirrors pngToICO in main.go.
function Save-Ico([hashtable[]]$frames, [string]$outFile) {
    $ms = New-Object System.IO.MemoryStream
    $bw = New-Object System.IO.BinaryWriter($ms)
    $bw.Write([uint16]0)                 # reserved
    $bw.Write([uint16]1)                 # type = icon
    $bw.Write([uint16]$frames.Count)
    $offset = 6 + 16 * $frames.Count
    foreach ($f in $frames) {
        $w = [int]$f.size
        $wb = if ($w -ge 256) { 0 } else { $w }   # 0 means 256 in the ICO spec
        $bw.Write([byte]$wb)             # width
        $bw.Write([byte]$wb)             # height
        $bw.Write([byte]0)               # palette count
        $bw.Write([byte]0)               # reserved
        $bw.Write([uint16]1)             # color planes
        $bw.Write([uint16]32)            # bits per pixel
        $bw.Write([uint32]$f.bytes.Length)
        $bw.Write([uint32]$offset)
        $offset += $f.bytes.Length
    }
    foreach ($f in $frames) { $bw.Write($f.bytes) }
    $bw.Flush()
    [System.IO.File]::WriteAllBytes($outFile, $ms.ToArray())
    $bw.Dispose(); $ms.Dispose()
    Write-Host "  wrote $outFile ($($frames.Count) frames)"
}

Write-Host "Generating tray PNGs..."
Save-MonoPng 16 ([System.Drawing.Color]::FromArgb(255, 255, 255, 255)) (Join-Path $here "tray_white16.png")
Save-MonoPng 32 ([System.Drawing.Color]::FromArgb(255, 255, 255, 255)) (Join-Path $here "tray_white32.png")
Save-MonoPng 16 ([System.Drawing.Color]::FromArgb(255, 0, 0, 0))       (Join-Path $here "tray_black16.png")
Save-MonoPng 32 ([System.Drawing.Color]::FromArgb(255, 0, 0, 0))       (Join-Path $here "tray_black32.png")

Write-Host "Generating colored Hotfix.ico..."
$icoFrames = @()
foreach ($sz in 16, 24, 32, 48, 64, 128, 256) {
    $bmp = Render-ColorBitmap $sz
    # Uncompressed BMP/DIB frames at every size — the most broadly compatible
    # ICO layout (read identically by goversioninfo, Inno Setup, and the shell).
    $icoFrames += @{ size = $sz; bytes = (Get-DibBytes $bmp) }
    $bmp.Dispose()
}
Save-Ico $icoFrames (Join-Path $here "Hotfix.ico")

Write-Host "Done."
