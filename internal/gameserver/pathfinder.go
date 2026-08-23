package gameserver

import (
	"container/heap"
	"math"
)

type geoNode struct {
	geoX, geoY, z int
	nswe          byte
	costG, costH  int
	costF         int
	parent        *geoNode
	index         int
}

func (n *geoNode) key() [3]int { return [3]int{n.geoX, n.geoY, n.z} }

func (n *geoNode) loc() GeoLoc {
	return GeoLoc{int32(WorldX(n.geoX)), int32(WorldY(n.geoY)), int32(n.z)}
}

type nodeHeap []*geoNode

func (h nodeHeap) Len() int           { return len(h) }
func (h nodeHeap) Less(i, j int) bool { return h[i].costF < h[j].costF }
func (h nodeHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}
func (h *nodeHeap) Push(x any) {
	n := x.(*geoNode)
	n.index = len(*h)
	*h = append(*h, n)
}
func (h *nodeHeap) Pop() any {
	old := *h
	n := old[len(old)-1]
	*h = old[:len(old)-1]
	n.index = -1
	return n
}

type pathFinder struct {
	eng    *GeoEngine
	opened nodeHeap
	have   map[[3]int]struct{}
	closed map[[3]int]struct{}
	gtx    int
	gty    int
	gtz    int
	cur    *geoNode
}

func newPathFinder(e *GeoEngine) *pathFinder {
	pf := &pathFinder{
		eng:    e,
		have:   map[[3]int]struct{}{},
		closed: map[[3]int]struct{}{},
	}
	heap.Init(&pf.opened)
	return pf
}

func (p *pathFinder) findPath(gox, goy, goz, gtx, gty, gtz int) []GeoLoc {
	p.gtx, p.gty, p.gtz = gtx, gty, gtz
	p.cur = &geoNode{geoX: gox, geoY: goy, z: goz, nswe: p.eng.NsweNearest(gox, goy, goz, nil)}
	p.cur.costG = 0
	p.cur.costH = p.costH(gox, goy, goz)
	p.cur.costF = p.cur.costH
	heap.Push(&p.opened, p.cur)
	p.have[p.cur.key()] = struct{}{}

	for count := 0; p.opened.Len() > 0 && count < p.eng.cfg.MaxIterations; count++ {
		p.cur = heap.Pop(&p.opened).(*geoNode)
		delete(p.have, p.cur.key())
		if p.cur.geoX == p.gtx && p.cur.geoY == p.gty && p.cur.z == p.gtz {
			return p.constructPath()
		}
		p.closed[p.cur.key()] = struct{}{}
		p.expand()
	}
	return nil
}

func (p *pathFinder) constructPath() []GeoLoc {
	var path []GeoLoc
	dx, dy := 0, 0
	parent := p.cur.parent
	for parent != nil {
		nx := parent.geoX - p.cur.geoX
		ny := parent.geoY - p.cur.geoY
		if dx != nx || dy != ny {
			path = append([]GeoLoc{p.cur.loc()}, path...)
			dx, dy = nx, ny
		}
		p.cur = parent
		parent = p.cur.parent
	}
	return path
}

func (p *pathFinder) expand() {
	nswe := p.cur.nswe
	if nswe == cellFlagNone {
		return
	}
	x, y := p.cur.geoX, p.cur.geoY
	z := p.cur.z + cellIgnoreHeight
	nsweN := p.addDirectional(x, y, z, nswe, 0, -1, cellFlagN)
	nsweS := p.addDirectional(x, y, z, nswe, 0, 1, cellFlagS)
	nsweW := p.addDirectional(x, y, z, nswe, -1, 0, cellFlagW)
	nsweE := p.addDirectional(x, y, z, nswe, 1, 0, cellFlagE)
	p.addCorner(x, y, z, -1, -1, cellFlagW, cellFlagN, nsweW, nsweN)
	p.addCorner(x, y, z, 1, -1, cellFlagE, cellFlagN, nsweE, nsweN)
	p.addCorner(x, y, z, -1, 1, cellFlagW, cellFlagS, nsweW, nsweS)
	p.addCorner(x, y, z, 1, 1, cellFlagE, cellFlagS, nsweE, nsweS)
}

func (p *pathFinder) addDirectional(x, y, z int, nswe byte, dx, dy int, flag byte) byte {
	if nswe&flag == 0 {
		return cellFlagNone
	}
	return p.addNode(x+dx, y+dy, z, false)
}

func (p *pathFinder) addCorner(x, y, z, dx, dy int, flagX, flagY, nsweX, nsweY byte) {
	if nsweX&flagY != 0 && nsweY&flagX != 0 {
		if p.nodeNswe(x+dx, y, z)&flagY != 0 {
			p.addNode(x+dx, y+dy, z, true)
		}
	}
}

func (p *pathFinder) nodeNswe(gx, gy, gz int) byte {
	if gx < 0 || gx >= geoCellsX || gy < 0 || gy >= geoCellsY {
		return cellFlagNone
	}
	block := p.eng.block(gx, gy)
	index := block.indexBelow(gx, gy, gz, nil)
	if index < 0 {
		return cellFlagNone
	}
	return block.nsweAt(index, nil)
}

func (p *pathFinder) addNode(gx, gy, gz int, diagonal bool) byte {
	if gx < 0 || gx >= geoCellsX || gy < 0 || gy >= geoCellsY {
		return cellFlagNone
	}
	block := p.eng.block(gx, gy)
	index := block.indexBelow(gx, gy, gz, nil)
	if index < 0 {
		return cellFlagNone
	}
	gz = int(block.heightAt(index, nil))
	nswe := block.nsweAt(index, nil)
	key := [3]int{gx, gy, gz}
	if _, ok := p.have[key]; ok {
		return nswe
	}
	if _, ok := p.closed[key]; ok {
		return nswe
	}
	weight := p.eng.cfg.MoveWeight
	if nswe == cellFlagAll {
		if diagonal {
			weight = p.eng.cfg.MoveWeightDiag
		}
	} else if diagonal {
		weight = p.eng.cfg.ObstacleWeightDiag
	} else {
		weight = p.eng.cfg.ObstacleWeight
	}
	n := &geoNode{geoX: gx, geoY: gy, z: gz, nswe: nswe, parent: p.cur}
	n.costG = weight
	if p.cur != nil {
		n.costG += p.cur.costG
	}
	n.costH = p.costH(gx, gy, gz)
	n.costF = n.costG + n.costH
	heap.Push(&p.opened, n)
	p.have[key] = struct{}{}
	return nswe
}

func (p *pathFinder) costH(gx, gy, gz int) int {
	dx := absInt(gx - p.gtx)
	dy := absInt(gy - p.gty)
	dz := absInt(gz-p.gtz) / cellHeight
	return int(math.Sqrt(float64(dx*dx+dy*dy+dz*dz)) * float64(p.eng.cfg.HeuristicWeight))
}
