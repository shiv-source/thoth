// Generates the extension toolbar icons as PNGs (Chrome/Firefox need raster
// icons; no binary assets live in the repo). A rounded blue square with a
// white "T", at the three sizes the manifests reference.
import { deflateSync } from 'node:zlib'

const crcTable = (() => {
    const table = new Uint32Array(256)
    for (let n = 0; n < 256; n++) {
        let c = n
        for (let k = 0; k < 8; k++) c = c & 1 ? 0xedb88320 ^ (c >>> 1) : c >>> 1
        table[n] = c >>> 0
    }
    return table
})()

function crc32(buf) {
    let c = 0xffffffff
    for (let i = 0; i < buf.length; i++) c = crcTable[(c ^ buf[i]) & 0xff] ^ (c >>> 8)
    return (c ^ 0xffffffff) >>> 0
}

function chunk(type, data) {
    const len = Buffer.alloc(4)
    len.writeUInt32BE(data.length, 0)
    const typeBuf = Buffer.from(type, 'ascii')
    const crc = Buffer.alloc(4)
    crc.writeUInt32BE(crc32(Buffer.concat([typeBuf, data])), 0)
    return Buffer.concat([len, typeBuf, data, crc])
}

function encodePNG(width, height, rgba) {
    const sig = Buffer.from([0x89, 0x50, 0x4e, 0x47, 0x0d, 0x0a, 0x1a, 0x0a])
    const ihdr = Buffer.alloc(13)
    ihdr.writeUInt32BE(width, 0)
    ihdr.writeUInt32BE(height, 4)
    ihdr[8] = 8 // bit depth
    ihdr[9] = 6 // color type RGBA
    const stride = width * 4
    const raw = Buffer.alloc((stride + 1) * height)
    for (let y = 0; y < height; y++) {
        raw[y * (stride + 1)] = 0 // filter: none
        rgba.copy(raw, y * (stride + 1) + 1, y * stride, y * stride + stride)
    }
    const idat = deflateSync(raw)
    return Buffer.concat([sig, chunk('IHDR', ihdr), chunk('IDAT', idat), chunk('IEND', Buffer.alloc(0))])
}

// roundedRectAlpha: 1 inside the rounded rectangle (corner radius size/4),
// 0 outside. Folds every point into the top-left octant for the test.
function roundedRectAlpha(x, y, size) {
    const r = size * 0.25
    const nx = Math.min(x + 0.5, size - (x + 0.5))
    const ny = Math.min(y + 0.5, size - (y + 0.5))
    const cx = Math.max(r, Math.min(size - r, nx))
    const cy = Math.max(r, Math.min(size - r, ny))
    return Math.hypot(nx - cx, ny - cy) <= r ? 1 : 0
}

// inGlyph: the white "T" — a top bar and a stem, in unit coordinates.
function inGlyph(x, y, size) {
    const u = (x + 0.5) / size
    const v = (y + 0.5) / size
    const topBar = v >= 0.14 && v <= 0.36 && u >= 0.18 && u <= 0.82
    const stem = v >= 0.14 && v <= 0.78 && u >= 0.44 && u <= 0.56
    return topBar || stem
}

export function generateIcon(size) {
    const px = Buffer.alloc(size * size * 4)
    for (let y = 0; y < size; y++) {
        for (let x = 0; x < size; x++) {
            const i = (y * size + x) * 4
            const alpha = roundedRectAlpha(x, y, size)
            if (inGlyph(x, y, size)) {
                px[i] = 255
                px[i + 1] = 255
                px[i + 2] = 255
                px[i + 3] = 255 * alpha
            } else {
                px[i] = 0x3b
                px[i + 1] = 0x82
                px[i + 2] = 0xf6
                px[i + 3] = 255 * alpha
            }
        }
    }
    return encodePNG(size, size, px)
}
