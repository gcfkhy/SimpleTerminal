const fs   = require('fs')
const path = require('path')
const { Resvg }    = require('@resvg/resvg-js')
const png2icons    = require('png2icons')

const svgPath = path.join(__dirname, '../build/icon.svg')
const pngPath = path.join(__dirname, '../build/appicon.png')
const icoPath = path.join(__dirname, '../build/windows/icon.ico')
const svg = fs.readFileSync(svgPath)

// 256x256 appicon.png（Wails 用）
const png256 = new Resvg(svg, { width: 256, height: 256 }).render().asPng()
fs.writeFileSync(pngPath, png256)
console.log('✓ build/appicon.png (256x256)')

// PNG → ICO（含 16/32/48/64/128/256 多尺寸）
const icoBuffer = png2icons.createICO(png256, png2icons.BILINEAR, 0, true)
if (!icoBuffer) throw new Error('ICO 生成返回空')
fs.writeFileSync(icoPath, icoBuffer)
console.log('✓ build/windows/icon.ico')
