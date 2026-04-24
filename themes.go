package main

import (
	"encoding/base64"
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"
)

type ThemeChar struct {
	Width  uint
	Height uint
	Data   string
}

type Theme struct {
	Name   string
	Width  uint
	Height uint
	Chars  map[string]ThemeChar
}

type ThemeManager struct {
	Themes map[string]*Theme
}

func NewThemeManager(themesDir string) *ThemeManager {
	tm := &ThemeManager{
		Themes: make(map[string]*Theme),
	}
	configs := []struct {
		name   string
		width  uint
		height uint
	}{
		{"moebooru", 45, 100},
		{"moebooru-h", 45, 100},
		{"number", 64, 64},
	}

	for _, cfg := range configs {
		theme := &Theme{
			Name:   cfg.name,
			Width:  cfg.width,
			Height: cfg.height,
			Chars:  make(map[string]ThemeChar),
		}
		themePath := filepath.Join(themesDir, cfg.name)
		imgFiles, err := os.ReadDir(themePath)

		if err != nil {
			fmt.Printf("Error reading theme folder %s: %v\n", cfg.name, err)
			continue
		}

		for _, imgF := range imgFiles {
			ext := strings.ToLower(filepath.Ext(imgF.Name()))
			char := strings.TrimSuffix(imgF.Name(), filepath.Ext(imgF.Name()))
			imgPath := filepath.Join(themePath, imgF.Name())
			data, err := os.ReadFile(imgPath)

			if err != nil {
				continue
			}

			mimeType := mime.TypeByExtension(ext)
			theme.Chars[char] = ThemeChar{
				Width:  cfg.width,
				Height: cfg.height,
				Data:   fmt.Sprintf("data:%s;base64,%s", mimeType, base64.StdEncoding.EncodeToString(data)),
			}
		}

		tm.Themes[cfg.name] = theme
	}

	return tm
}

func (tm *ThemeManager) GenerateSVG(count int64, themeName string, scale float64, padding int) string {
	theme := tm.Themes[themeName]
	countStr := fmt.Sprintf("%0*d", padding, count)
	chars := strings.Split(countStr, "")

	var defs strings.Builder
	var parts strings.Builder
	var maxX, maxY float64
	var currentX float64

	uniqueChars := make(map[string]bool)

	for _, c := range chars {
		uniqueChars[c] = true
	}

	for c := range uniqueChars {
		charData := theme.Chars[c]
		w := float64(charData.Width) * scale
		h := float64(charData.Height) * scale

		if h > maxY {
			maxY = h
		}

		fmt.Fprintf(&defs, "\n<image id=\"%s\" width=\"%.2f\" height=\"%.2f\" xlink:href=\"%s\" />", c, w, h, charData.Data)
	}

	for _, c := range chars {
		charData := theme.Chars[c]
		w := float64(charData.Width) * scale
		fmt.Fprintf(&parts, "\n<use x=\"%.2f\" xlink:href=\"#%s\" />", currentX, c)
		currentX += w
	}

	maxX = currentX
	style := `
	svg {
		image-rendering: pixelated;
		image-rendering: crisp-edges;
		filter: brightness(.5);
	}
	`

	return fmt.Sprintf(`
	<svg viewBox="0 0 %.2f %.2f" width="%.2f" height="%.2f" version="1.1" xmlns="http://www.w3.org/2000/svg" xmlns:xlink="http://www.w3.org/1999/xlink">
		<title>Counter Badge</title>
		<style>%s</style>
		<defs>%s</defs>
		<g>%s</g>
	</svg>
	`, maxX, maxY, maxX, maxY, style, defs.String(), parts.String())
}
