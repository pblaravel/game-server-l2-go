package gameserver

import (
	"encoding/binary"
	"fmt"
)

// geoBlock is Java geodata.ABlock.
type geoBlock interface {
	hasGeoPos() bool
	heightNearest(geoX, geoY, worldZ int, ignore GeoObject) int16
	nsweNearest(geoX, geoY, worldZ int, ignore GeoObject) byte
	indexNearest(geoX, geoY, worldZ int, ignore GeoObject) int
	indexAbove(geoX, geoY, worldZ int, ignore GeoObject) int
	indexBelow(geoX, geoY, worldZ int, ignore GeoObject) int
	heightAt(index int, ignore GeoObject) int16
	nsweAt(index int, ignore GeoObject) byte
}

type geoDynamic interface {
	geoBlock
	addGeoObject(object GeoObject)
	removeGeoObject(object GeoObject)
}

type blockNull struct{}

func (b *blockNull) hasGeoPos() bool { return false }
func (b *blockNull) heightNearest(_, _, worldZ int, _ GeoObject) int16 {
	return int16(worldZ)
}
func (b *blockNull) nsweNearest(_, _, _ int, _ GeoObject) byte { return cellFlagAll }
func (b *blockNull) indexNearest(_, _, _ int, _ GeoObject) int { return 0 }
func (b *blockNull) indexAbove(_, _, _ int, _ GeoObject) int   { return 0 }
func (b *blockNull) indexBelow(_, _, _ int, _ GeoObject) int   { return 0 }
func (b *blockNull) heightAt(_ int, _ GeoObject) int16         { return 0 }
func (b *blockNull) nsweAt(_ int, _ GeoObject) byte            { return cellFlagAll }

type blockFlat struct {
	height int16
	nswe   byte
}

func readBlockFlat(raw []byte, off int, typ GeoType) (geoBlock, int, error) {
	need := 2
	if typ == GeoL2OFF {
		need = 4
	}
	if off+need > len(raw) {
		return nil, off, fmt.Errorf("truncated flat block")
	}
	h := int16(binary.LittleEndian.Uint16(raw[off:]))
	off += 2
	if typ == GeoL2OFF {
		off += 2
	}
	return &blockFlat{height: h, nswe: cellFlagAll}, off, nil
}

func (b *blockFlat) hasGeoPos() bool                              { return true }
func (b *blockFlat) heightNearest(_, _, _ int, _ GeoObject) int16 { return b.height }
func (b *blockFlat) nsweNearest(_, _, _ int, _ GeoObject) byte    { return b.nswe }
func (b *blockFlat) indexNearest(_, _, _ int, _ GeoObject) int    { return 0 }
func (b *blockFlat) indexAbove(_, _, worldZ int, _ GeoObject) int {
	if int(b.height) > worldZ {
		return 0
	}
	return -1
}
func (b *blockFlat) indexBelow(_, _, worldZ int, _ GeoObject) int {
	if int(b.height) < worldZ {
		return 0
	}
	return -1
}
func (b *blockFlat) heightAt(_ int, _ GeoObject) int16 { return b.height }
func (b *blockFlat) nsweAt(_ int, _ GeoObject) byte    { return b.nswe }

type blockComplex struct {
	buffer []byte
}

func complexIndex(geoX, geoY int) int {
	return ((geoX%blockCellsX)*blockCellsY + (geoY % blockCellsY)) * 3
}

func readBlockComplex(raw []byte, off int) (geoBlock, int, error) {
	need := blockCells * 2
	if off+need > len(raw) {
		return nil, off, fmt.Errorf("truncated complex block")
	}
	buf := make([]byte, blockCells*3)
	for i := 0; i < blockCells; i++ {
		data := int16(binary.LittleEndian.Uint16(raw[off:]))
		off += 2
		nswe, h := unpackGeoShort(data)
		buf[i*3] = nswe
		putCellHeight(buf, i*3, h)
	}
	return &blockComplex{buffer: buf}, off, nil
}

func (b *blockComplex) hasGeoPos() bool { return true }

func (b *blockComplex) heightNearest(geoX, geoY, _ int, _ GeoObject) int16 {
	return cellHeightAt(b.buffer, complexIndex(geoX, geoY))
}

func (b *blockComplex) nsweNearest(geoX, geoY, _ int, _ GeoObject) byte {
	return b.buffer[complexIndex(geoX, geoY)]
}

func (b *blockComplex) indexNearest(geoX, geoY, _ int, _ GeoObject) int {
	return complexIndex(geoX, geoY)
}

func (b *blockComplex) indexAbove(geoX, geoY, worldZ int, _ GeoObject) int {
	idx := complexIndex(geoX, geoY)
	if int(cellHeightAt(b.buffer, idx)) > worldZ {
		return idx
	}
	return -1
}

func (b *blockComplex) indexBelow(geoX, geoY, worldZ int, _ GeoObject) int {
	idx := complexIndex(geoX, geoY)
	if int(cellHeightAt(b.buffer, idx)) < worldZ {
		return idx
	}
	return -1
}

func (b *blockComplex) heightAt(index int, _ GeoObject) int16 {
	return cellHeightAt(b.buffer, index)
}

func (b *blockComplex) nsweAt(index int, _ GeoObject) byte { return b.buffer[index] }

type blockMultilayer struct {
	buffer []byte
}

func readBlockMultilayer(raw []byte, off int, typ GeoType, temp []byte) (geoBlock, int, []byte, error) {
	temp = temp[:0]
	for cell := 0; cell < blockCells; cell++ {
		var layers int
		if typ != GeoL2OFF {
			if off >= len(raw) {
				return nil, off, temp, fmt.Errorf("truncated multilayer layer count")
			}
			layers = int(int8(raw[off]))
			off++
		} else {
			if off+2 > len(raw) {
				return nil, off, temp, fmt.Errorf("truncated multilayer L2OFF layer count")
			}
			layers = int(int8(int16(binary.LittleEndian.Uint16(raw[off:]))))
			off += 2
		}
		if layers <= 0 || layers > maxLayers {
			return nil, off, temp, fmt.Errorf("invalid layer count %d", layers)
		}
		temp = append(temp, byte(layers))
		for layer := 0; layer < layers; layer++ {
			if off+2 > len(raw) {
				return nil, off, temp, fmt.Errorf("truncated multilayer cell")
			}
			data := int16(binary.LittleEndian.Uint16(raw[off:]))
			off += 2
			nswe, h := unpackGeoShort(data)
			temp = append(temp, nswe, byte(h), byte(h>>8))
		}
	}
	buf := make([]byte, len(temp))
	copy(buf, temp)
	return &blockMultilayer{buffer: buf}, off, temp, nil
}

func multilayerCellStart(buf []byte, geoX, geoY int) int {
	index := 0
	cells := (geoX%blockCellsX)*blockCellsY + (geoY % blockCellsY)
	for i := 0; i < cells; i++ {
		index += int(buf[index])*3 + 1
	}
	return index
}

func (b *blockMultilayer) hasGeoPos() bool { return true }

func (b *blockMultilayer) heightNearest(geoX, geoY, worldZ int, ignore GeoObject) int16 {
	return cellHeightAt(b.buffer, b.indexNearest(geoX, geoY, worldZ, ignore))
}

func (b *blockMultilayer) nsweNearest(geoX, geoY, worldZ int, ignore GeoObject) byte {
	return b.buffer[b.indexNearest(geoX, geoY, worldZ, ignore)]
}

func (b *blockMultilayer) indexNearest(geoX, geoY, worldZ int, _ GeoObject) int {
	index := multilayerCellStart(b.buffer, geoX, geoY)
	layers := int(int8(b.buffer[index]))
	index++
	limit := int(^uint(0) >> 1)
	for layers > 0 {
		layers--
		height := int(cellHeightAt(b.buffer, index))
		distance := height - worldZ
		if distance < 0 {
			distance = -distance
		}
		if distance > limit {
			break
		}
		limit = distance
		index += 3
	}
	return index - 3
}

func (b *blockMultilayer) indexAbove(geoX, geoY, worldZ int, _ GeoObject) int {
	index := multilayerCellStart(b.buffer, geoX, geoY)
	layers := int(int8(b.buffer[index]))
	index++
	index += (layers - 1) * 3
	for layers > 0 {
		layers--
		if int(cellHeightAt(b.buffer, index)) > worldZ {
			return index
		}
		index -= 3
	}
	return -1
}

func (b *blockMultilayer) indexBelow(geoX, geoY, worldZ int, _ GeoObject) int {
	index := multilayerCellStart(b.buffer, geoX, geoY)
	layers := int(int8(b.buffer[index]))
	index++
	for layers > 0 {
		layers--
		if int(cellHeightAt(b.buffer, index)) < worldZ {
			return index
		}
		index += 3
	}
	return -1
}

func (b *blockMultilayer) heightAt(index int, _ GeoObject) int16 {
	return cellHeightAt(b.buffer, index)
}

func (b *blockMultilayer) nsweAt(index int, _ GeoObject) byte { return b.buffer[index] }

func geoObjectContains(list []GeoObject, o GeoObject) bool {
	for _, x := range list {
		if x == o {
			return true
		}
	}
	return false
}
