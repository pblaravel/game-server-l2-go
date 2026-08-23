package gameserver

type blockComplexDynamic struct {
	blockComplex
	bx, by   int
	original []byte
	objects  []GeoObject
}

func newComplexDynamicFromFlat(bx, by int, flat *blockFlat) *blockComplexDynamic {
	buf := make([]byte, blockCells*3)
	orig := make([]byte, blockCells*3)
	lo := byte(flat.height)
	hi := byte(flat.height >> 8)
	for i := 0; i < blockCells; i++ {
		buf[i*3] = flat.nswe
		buf[i*3+1] = lo
		buf[i*3+2] = hi
	}
	copy(orig, buf)
	return &blockComplexDynamic{blockComplex: blockComplex{buffer: buf}, bx: bx, by: by, original: orig}
}

func newComplexDynamicFromComplex(bx, by int, src *blockComplex) *blockComplexDynamic {
	orig := make([]byte, len(src.buffer))
	copy(orig, src.buffer)
	return &blockComplexDynamic{blockComplex: blockComplex{buffer: src.buffer}, bx: bx, by: by, original: orig}
}

func (b *blockComplexDynamic) pick(ignore GeoObject) []byte {
	if geoObjectContains(b.objects, ignore) {
		return b.original
	}
	return b.buffer
}

func (b *blockComplexDynamic) heightNearest(geoX, geoY, _ int, ignore GeoObject) int16 {
	return cellHeightAt(b.pick(ignore), complexIndex(geoX, geoY))
}

func (b *blockComplexDynamic) nsweNearest(geoX, geoY, _ int, ignore GeoObject) byte {
	return b.pick(ignore)[complexIndex(geoX, geoY)]
}

func (b *blockComplexDynamic) indexAbove(geoX, geoY, worldZ int, ignore GeoObject) int {
	idx := complexIndex(geoX, geoY)
	if int(cellHeightAt(b.pick(ignore), idx)) > worldZ {
		return idx
	}
	return -1
}

func (b *blockComplexDynamic) indexBelow(geoX, geoY, worldZ int, ignore GeoObject) int {
	idx := complexIndex(geoX, geoY)
	if int(cellHeightAt(b.pick(ignore), idx)) < worldZ {
		return idx
	}
	return -1
}

func (b *blockComplexDynamic) heightAt(index int, ignore GeoObject) int16 {
	return cellHeightAt(b.pick(ignore), index)
}

func (b *blockComplexDynamic) nsweAt(index int, ignore GeoObject) byte {
	return b.pick(ignore)[index]
}

func (b *blockComplexDynamic) addGeoObject(object GeoObject) {
	if geoObjectContains(b.objects, object) {
		return
	}
	b.objects = append(b.objects, object)
	b.update()
}

func (b *blockComplexDynamic) removeGeoObject(object GeoObject) {
	for i, o := range b.objects {
		if o == object {
			b.objects = append(b.objects[:i], b.objects[i+1:]...)
			b.update()
			return
		}
	}
}

func (b *blockComplexDynamic) update() {
	copy(b.buffer, b.original)
	minBX := b.bx * blockCellsX
	minBY := b.by * blockCellsY
	maxBX := minBX + blockCellsX
	maxBY := minBY + blockCellsY
	for _, object := range b.objects {
		minOX, minOY := object.GeoX(), object.GeoY()
		minOZ := object.GeoZ()
		maxOZ := minOZ + object.Height()
		geoData := object.ObjectGeoData()
		if len(geoData) == 0 || len(geoData[0]) == 0 {
			continue
		}
		minGX := max(minBX, minOX)
		minGY := max(minBY, minOY)
		maxGX := min(maxBX, minOX+len(geoData))
		maxGY := min(maxBY, minOY+len(geoData[0]))
		for gx := minGX; gx < maxGX; gx++ {
			for gy := minGY; gy < maxGY; gy++ {
				objNswe := geoData[gx-minOX][gy-minOY]
				if objNswe == cellFlagAll {
					continue
				}
				ib := ((gx-minBX)*blockCellsY + (gy - minBY)) * 3
				if b.buffer[ib+1] != b.original[ib+1] || b.buffer[ib+2] != b.original[ib+2] {
					continue
				}
				if objNswe == cellFlagNone {
					b.buffer[ib] = cellFlagNone
					putCellHeight(b.buffer, ib, int16(maxOZ))
					continue
				}
				z := int(b.heightAt(ib, nil))
				diff := z - minOZ
				if diff < 0 {
					diff = -diff
				}
				if diff > cellIgnoreHeight {
					continue
				}
				b.buffer[ib] &= objNswe
			}
		}
	}
}

type blockMultilayerDynamic struct {
	blockMultilayer
	bx, by   int
	original []byte
	objects  []GeoObject
}

func newMultilayerDynamic(bx, by int, src *blockMultilayer) *blockMultilayerDynamic {
	orig := make([]byte, len(src.buffer))
	copy(orig, src.buffer)
	return &blockMultilayerDynamic{blockMultilayer: blockMultilayer{buffer: src.buffer}, bx: bx, by: by, original: orig}
}

func (b *blockMultilayerDynamic) pick(ignore GeoObject) []byte {
	if geoObjectContains(b.objects, ignore) {
		return b.original
	}
	return b.buffer
}

func (b *blockMultilayerDynamic) heightNearest(geoX, geoY, worldZ int, ignore GeoObject) int16 {
	return cellHeightAt(b.pick(ignore), b.indexNearest(geoX, geoY, worldZ, ignore))
}

func (b *blockMultilayerDynamic) nsweNearest(geoX, geoY, worldZ int, ignore GeoObject) byte {
	return b.pick(ignore)[b.indexNearest(geoX, geoY, worldZ, ignore)]
}

func (b *blockMultilayerDynamic) indexNearest(geoX, geoY, worldZ int, ignore GeoObject) int {
	buf := b.pick(ignore)
	index := multilayerCellStart(buf, geoX, geoY)
	layers := int(int8(buf[index]))
	index++
	limit := int(^uint(0) >> 1)
	for layers > 0 {
		layers--
		height := int(cellHeightAt(buf, index))
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

func (b *blockMultilayerDynamic) indexAbove(geoX, geoY, worldZ int, ignore GeoObject) int {
	buf := b.pick(ignore)
	index := multilayerCellStart(buf, geoX, geoY)
	layers := int(int8(buf[index]))
	index++
	index += (layers - 1) * 3
	for layers > 0 {
		layers--
		if int(cellHeightAt(buf, index)) > worldZ {
			return index
		}
		index -= 3
	}
	return -1
}

func (b *blockMultilayerDynamic) indexBelow(geoX, geoY, worldZ int, ignore GeoObject) int {
	buf := b.pick(ignore)
	index := multilayerCellStart(buf, geoX, geoY)
	layers := int(int8(buf[index]))
	index++
	for layers > 0 {
		layers--
		if int(cellHeightAt(buf, index)) < worldZ {
			return index
		}
		index += 3
	}
	return -1
}

func (b *blockMultilayerDynamic) heightAt(index int, ignore GeoObject) int16 {
	return cellHeightAt(b.pick(ignore), index)
}

func (b *blockMultilayerDynamic) nsweAt(index int, ignore GeoObject) byte {
	return b.pick(ignore)[index]
}

func (b *blockMultilayerDynamic) addGeoObject(object GeoObject) {
	if geoObjectContains(b.objects, object) {
		return
	}
	b.objects = append(b.objects, object)
	b.update()
}

func (b *blockMultilayerDynamic) removeGeoObject(object GeoObject) {
	for i, o := range b.objects {
		if o == object {
			b.objects = append(b.objects[:i], b.objects[i+1:]...)
			b.update()
			return
		}
	}
}

func (b *blockMultilayerDynamic) update() {
	copy(b.buffer, b.original)
	minBX := b.bx * blockCellsX
	minBY := b.by * blockCellsY
	maxBX := minBX + blockCellsX
	maxBY := minBY + blockCellsY
	for _, object := range b.objects {
		minOX, minOY := object.GeoX(), object.GeoY()
		minOZ := object.GeoZ()
		maxOZ := minOZ + object.Height()
		geoData := object.ObjectGeoData()
		if len(geoData) == 0 || len(geoData[0]) == 0 {
			continue
		}
		minGX := max(minBX, minOX)
		minGY := max(minBY, minOY)
		maxGX := min(maxBX, minOX+len(geoData))
		maxGY := min(maxBY, minOY+len(geoData[0]))
		for gx := minGX; gx < maxGX; gx++ {
			for gy := minGY; gy < maxGY; gy++ {
				objNswe := geoData[gx-minOX][gy-minOY]
				if objNswe == cellFlagAll {
					continue
				}
				ib := b.indexNearest(gx, gy, minOZ, nil)
				if b.buffer[ib+1] != b.original[ib+1] || b.buffer[ib+2] != b.original[ib+2] {
					continue
				}
				if objNswe == cellFlagNone {
					b.buffer[ib] = cellFlagNone
					z := maxOZ
					if i := b.indexAbove(gx, gy, minOZ, nil); i != -1 {
						az := int(b.heightAt(i, nil))
						if az <= maxOZ {
							z = az - cellIgnoreHeight
						}
					}
					putCellHeight(b.buffer, ib, int16(z))
					continue
				}
				z := int(b.heightAt(ib, nil))
				diff := z - minOZ
				if diff < 0 {
					diff = -diff
				}
				if diff > cellIgnoreHeight {
					continue
				}
				b.buffer[ib] &= objNswe
			}
		}
	}
}
