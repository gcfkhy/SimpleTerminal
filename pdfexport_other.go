//go:build !windows

package main

import "errors"

func printHTMLToPDF(html, outPath string, pageWIn, pageHIn, scale float64) error {
	return errors.New("PDF 导出仅支持 Windows")
}
