package main

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
)

type StageProps struct {
	roundpos bool
}

func newStageProps() StageProps {
	sp := StageProps{
		roundpos: false,
	}

	return sp
}

type EnvShake struct {
	time  int32
	freq  float32
	ampl  float32
	phase float32
	mul   float32
}

func (es *EnvShake) clear() {
	*es = EnvShake{freq: float32(math.Pi / 3), ampl: -4.0,
		phase: float32(math.NaN()), mul: 1.0}
}
func (es *EnvShake) setDefPhase() {
	if math.IsNaN(float64(es.phase)) {
		if es.freq >= math.Pi/2 {
			es.phase = math.Pi / 2
		} else {
			es.phase = 0
		}
	}
}
func (es *EnvShake) next() {
	if es.time > 0 {
		es.time--
		es.phase += es.freq
		es.ampl *= es.mul
	}
}
func (es *EnvShake) getOffset() float32 {
	if es.time > 0 {
		return es.ampl * float32(math.Sin(float64(es.phase)))
	}
	return 0
}

type BgcType int32

const (
	BT_Null BgcType = iota
	BT_Anim
	BT_Visible
	BT_Enable
	BT_PalFX
	BT_PosSet
	BT_PosAdd
	BT_RemapPal
	BT_SinX
	BT_SinY
	BT_VelSet
	BT_VelAdd
)

type bgAction struct {
	offset      [2]float32
	sinoffset   [2]float32
	pos, vel    [2]float32
	radius      [2]float32
	sintime     [2]int32
	sinlooptime [2]int32
}

func (bga *bgAction) clear() {
	*bga = bgAction{}
}
func (bga *bgAction) action() {
	for i := 0; i < 2; i++ {
		bga.pos[i] += bga.vel[i]
		if bga.sinlooptime[i] > 0 {
			bga.sinoffset[i] = bga.radius[i] * float32(math.Sin(
				2*math.Pi*float64(bga.sintime[i])/float64(bga.sinlooptime[i])))
			bga.sintime[i]++
			if bga.sintime[i] >= bga.sinlooptime[i] {
				bga.sintime[i] = 0
			}
		} else {
			bga.sinoffset[i] = 0
		}
		bga.offset[i] = bga.pos[i] + bga.sinoffset[i]
	}
}

type backGround struct {
	typ                int
	palfx              *PalFX
	anim               Animation
	bga                bgAction
	id                 int32
	start              [2]float32
	xofs               float32
	camstartx          float32
	delta              [2]float32
	width              [2]int32
	xscale             [2]float32
	rasterx            [2]float32
	yscalestart        float32
	yscaledelta        float32
	actionno           int32
	startv             [2]float32
	startrad           [2]float32
	startsint          [2]int32
	startsinlt         [2]int32
	visible            bool
	active             bool
	positionlink       bool
	toplayer           bool
	autoresizeparallax bool
	notmaskwindow      int32
	startrect          [4]int32
	windowdelta        [2]float32
	scalestart         [2]float32
	scaledelta         [2]float32
	zoomdelta          [2]float32
	zoomscaledelta     [2]float32
	xbottomzoomdelta   float32
	roundpos           bool
}

func newBackGround(sff *Sff) *backGround {
	return &backGround{palfx: newPalFX(), anim: *newAnimation(sff), delta: [...]float32{1, 1}, zoomdelta: [...]float32{1, math.MaxFloat32},
		xscale: [...]float32{1, 1}, rasterx: [...]float32{1, 1}, yscalestart: 100, scalestart: [...]float32{1, 1}, xbottomzoomdelta: math.MaxFloat32,
		zoomscaledelta: [...]float32{math.MaxFloat32, math.MaxFloat32}, actionno: -1, visible: true, active: true, autoresizeparallax: false,
		startrect: [...]int32{-32768, -32768, 65535, 65535}}
}
func readBackGround(is IniSection, link *backGround,
	sff *Sff, at AnimationTable, camstartx float32, sProps StageProps) *backGround {
	bg := newBackGround(sff)
	bg.camstartx = camstartx
	typ := is["type"]
	if len(typ) == 0 {
		return bg
	}
	switch typ[0] {
	case 'N', 'n':
		bg.typ = 0 // normal
	case 'A', 'a':
		bg.typ = 1 // anim
	case 'P', 'p':
		bg.typ = 2 // parallax
	case 'D', 'd':
		bg.typ = 3 // dummy
	default:
		return bg
	}
	var tmp int32
	if is.ReadI32("layerno", &tmp) {
		bg.toplayer = tmp == 1
		if tmp < 0 || tmp > 1 {
			bg.typ = 3
		}
	}
	if bg.typ != 3 {
		var hasAnim bool
		if (bg.typ != 0 || len(is["spriteno"]) == 0) &&
			is.ReadI32("actionno", &bg.actionno) {
			if a := at.get(bg.actionno); a != nil {
				bg.anim = *a
				hasAnim = true
			}
		}
		if hasAnim {
			if bg.typ == 0 {
				bg.typ = 1
			}
		} else {
			var g, n int32
			if is.readI32ForStage("spriteno", &g, &n) {
				bg.anim.frames = []AnimFrame{*newAnimFrame()}
				bg.anim.frames[0].Group, bg.anim.frames[0].Number =
					I32ToI16(g), I32ToI16(n)
			}
			if is.ReadI32("mask", &tmp) {
				if tmp != 0 {
					bg.anim.mask = 0
				} else {
					bg.anim.mask = -1
				}
			}
		}
	}
	is.ReadBool("positionlink", &bg.positionlink)
	if bg.positionlink && link != nil {
		bg.startv = link.startv
		bg.delta = link.delta
	}
	is.ReadBool("autoresizeparallax", &bg.autoresizeparallax)
	is.readF32ForStage("start", &bg.start[0], &bg.start[1])
	if !bg.positionlink {
		is.readF32ForStage("delta", &bg.delta[0], &bg.delta[1])
	}
	is.readF32ForStage("scalestart", &bg.scalestart[0], &bg.scalestart[1])
	is.readF32ForStage("scaledelta", &bg.scaledelta[0], &bg.scaledelta[1])
	is.readF32ForStage("xbottomzoomdelta", &bg.xbottomzoomdelta)
	is.readF32ForStage("zoomscaledelta", &bg.zoomscaledelta[0], &bg.zoomscaledelta[1])
	is.readF32ForStage("zoomdelta", &bg.zoomdelta[0], &bg.zoomdelta[1])
	if bg.zoomdelta[0] != math.MaxFloat32 && bg.zoomdelta[1] == math.MaxFloat32 {
		bg.zoomdelta[1] = bg.zoomdelta[0]
	}
	switch strings.ToLower(is["trans"]) {
	case "add":
		bg.anim.mask = 0
		bg.anim.srcAlpha = 255
		bg.anim.dstAlpha = 255
		s, d := int32(bg.anim.srcAlpha), int32(bg.anim.dstAlpha)
		if is.readI32ForStage("alpha", &s, &d) {
			bg.anim.srcAlpha = int16(Clamp(s, 0, 255))
			bg.anim.dstAlpha = int16(Clamp(d, 0, 255))
			if bg.anim.srcAlpha == 1 && bg.anim.dstAlpha == 255 {
				bg.anim.srcAlpha = 0
			}
		}
	case "add1":
		bg.anim.mask = 0
		bg.anim.srcAlpha = 255
		bg.anim.dstAlpha = ^255
		var s, d int32 = 255, 255
		if is.readI32ForStage("alpha", &s, &d) {
			bg.anim.srcAlpha = int16(Min(255, s))
			//bg.anim.dstAlpha = ^int16(Clamp(d, 0, 255))
			bg.anim.dstAlpha = int16(Clamp(d, 0, 255))
		}
	case "addalpha":
		bg.anim.mask = 0
		s, d := int32(bg.anim.srcAlpha), int32(bg.anim.dstAlpha)
		if is.readI32ForStage("alpha", &s, &d) {
			bg.anim.srcAlpha = int16(Clamp(s, 0, 255))
			bg.anim.dstAlpha = int16(Clamp(d, 0, 255))
			if bg.anim.srcAlpha == 1 && bg.anim.dstAlpha == 255 {
				bg.anim.srcAlpha = 0
			}
		}
	case "sub":
		bg.anim.mask = 0
		bg.anim.srcAlpha = 1
		bg.anim.dstAlpha = 255
	case "none":
		bg.anim.srcAlpha = -1
		bg.anim.dstAlpha = 0
	}
	if is.readI32ForStage("tile", &bg.anim.tile.x, &bg.anim.tile.y) {
		if bg.typ == 2 {
			bg.anim.tile.y = 0
		}
		if bg.anim.tile.x < 0 {
			bg.anim.tile.x = math.MaxInt32
		}
	}
	if bg.typ == 2 {
		if !is.readI32ForStage("width", &bg.width[0], &bg.width[1]) {
			is.readF32ForStage("xscale", &bg.rasterx[0], &bg.rasterx[1])
		}
		is.ReadF32("yscalestart", &bg.yscalestart)
		is.ReadF32("yscaledelta", &bg.yscaledelta)
	} else {
		is.ReadI32("tilespacing", &bg.anim.tile.sx, &bg.anim.tile.sy)
		//bg.anim.tile.sy = bg.anim.tile.sx
		if bg.actionno < 0 && len(bg.anim.frames) > 0 {
			if spr := sff.GetSprite(
				bg.anim.frames[0].Group, bg.anim.frames[0].Number); spr != nil {
				bg.anim.tile.sx += int32(spr.Size[0])
				bg.anim.tile.sy += int32(spr.Size[1])
			}
		} else {
			if bg.anim.tile.sx == 0 {
				bg.anim.tile.x = 0
			}
			if bg.anim.tile.sy == 0 {
				bg.anim.tile.y = 0
			}
		}
	}
	if is.readI32ForStage("window", &bg.startrect[0], &bg.startrect[1],
		&bg.startrect[2], &bg.startrect[3]) {
		bg.startrect[2] = Max(0, bg.startrect[2]+1-bg.startrect[0])
		bg.startrect[3] = Max(0, bg.startrect[3]+1-bg.startrect[1])
		bg.notmaskwindow = 1
	}
	if is.readI32ForStage("maskwindow", &bg.startrect[0], &bg.startrect[1],
		&bg.startrect[2], &bg.startrect[3]) {
		bg.startrect[2] = Max(0, bg.startrect[2]-bg.startrect[0])
		bg.startrect[3] = Max(0, bg.startrect[3]-bg.startrect[1])
		bg.notmaskwindow = 0
	}
	is.readF32ForStage("windowdelta", &bg.windowdelta[0], &bg.windowdelta[1])
	is.ReadI32("id", &bg.id)
	is.readF32ForStage("velocity", &bg.startv[0], &bg.startv[1])
	for i := 0; i < 2; i++ {
		var name string
		if i == 0 {
			name = "sin.x"
		} else {
			name = "sin.y"
		}
		r, slt, st := float32(math.NaN()), float32(math.NaN()), float32(math.NaN())
		if is.readF32ForStage(name, &r, &slt, &st) {
			if !math.IsNaN(float64(r)) {
				bg.startrad[i], bg.bga.radius[i] = r, r
			}
			if !math.IsNaN(float64(slt)) {
				var slti int32
				is.readI32ForStage(name, &tmp, &slti)
				bg.startsinlt[i], bg.bga.sinlooptime[i] = slti, slti
			}
			if bg.bga.sinlooptime[i] > 0 && !math.IsNaN(float64(st)) {
				bg.bga.sintime[i] = int32(st*float32(bg.bga.sinlooptime[i])/360) %
					bg.bga.sinlooptime[i]
				if bg.bga.sintime[i] < 0 {
					bg.bga.sintime[i] += bg.bga.sinlooptime[i]
				}
				bg.startsint[i] = bg.bga.sintime[i]
			}
		}
	}
	if !is.ReadBool("roundpos", &bg.roundpos) {
		bg.roundpos = sProps.roundpos
	}
	return bg
}
func (bg *backGround) reset() {
	bg.palfx.clear()
	bg.anim.Reset()
	bg.bga.clear()
	bg.bga.vel = bg.startv
	bg.bga.radius = bg.startrad
	bg.bga.sintime = bg.startsint
	bg.bga.sinlooptime = bg.startsinlt
	bg.palfx.time = -1
}
func (bg backGround) draw(pos [2]float32, scl, bgscl, lclscl float32,
	stgscl [2]float32, shakeY float32, isStage bool) {
	if bg.typ == 2 && (bg.width[0] != 0 || bg.width[1] != 0) && bg.anim.spr != nil {
		bg.xscale[0] = float32(bg.width[0]) / float32(bg.anim.spr.Size[0])
		bg.xscale[1] = float32(bg.width[1]) / float32(bg.anim.spr.Size[0])
		bg.xofs = -float32(bg.width[0])/2 + float32(bg.anim.spr.Offset[0])*bg.xscale[0]
	}
	xras := (bg.rasterx[1] - bg.rasterx[0]) / bg.rasterx[0]
	xbs, dx := bg.xscale[1], MaxF(0, bg.delta[0]*bgscl)
	var sclx_recip, sclx, scly float32 = 1, 1, 1
	lscl := [...]float32{lclscl * stgscl[0], lclscl * stgscl[1]}
	if bg.zoomdelta[0] != math.MaxFloat32 {
		sclx = scl + (1-scl)*(1-bg.zoomdelta[0])
		scly = scl + (1-scl)*(1-bg.zoomdelta[1])
		if !bg.autoresizeparallax {
			sclx_recip = (1 + bg.zoomdelta[0]*((1/(sclx*lscl[0])*lscl[0])-1))
		}
	} else {
		sclx = MaxF(0, scl+(1-scl)*(1-dx))
		scly = MaxF(0, scl+(1-scl)*(1-MaxF(0, bg.delta[1]*bgscl)))
	}
	if sclx != 0 && bg.autoresizeparallax {
		tmp := 1 / sclx
		if bg.xbottomzoomdelta != math.MaxFloat32 {
			xbs *= MaxF(0, scl+(1-scl)*(1-bg.xbottomzoomdelta*(xbs/bg.xscale[0]))) * tmp
		} else {
			xbs *= MaxF(0, scl+(1-scl)*(1-dx*(xbs/bg.xscale[0]))) * tmp
		}
		tmp *= MaxF(0, scl+(1-scl)*(1-dx*(xras+1)))
		xras -= tmp - 1
		xbs *= tmp
	}
	var xs3, ys3 float32 = 1, 1
	if bg.zoomscaledelta[0] != math.MaxFloat32 {
		xs3 = ((scl + (1-scl)*(1-bg.zoomscaledelta[0])) / sclx)
	}
	if bg.zoomscaledelta[1] != math.MaxFloat32 {
		ys3 = ((scl + (1-scl)*(1-bg.zoomscaledelta[1])) / scly)
	}
	scly *= lclscl
	sclx *= lscl[0]
	// This handles the flooring of the camera position in MUGEN versions earlier than 1.0.
	if bg.roundpos {
		for i := 0; i < 2; i++ {
			pos[i] = float32(math.Floor(float64(pos[i])))
		}
	}
	x := bg.start[0] + bg.xofs - (pos[0]/stgscl[0]+bg.camstartx)*bg.delta[0] +
		bg.bga.offset[0]
	zoomybound := sys.cam.CameraZoomYBound * float32(Btoi(isStage))
	yScrollPos := ((pos[1] - (zoomybound / lclscl)) / stgscl[1]) * bg.delta[1] * bgscl
	yScrollPos += ((zoomybound / lclscl) / stgscl[1]) * Pow(bg.zoomdelta[1], 1.4) / bgscl
	y := bg.start[1] - yScrollPos + bg.bga.offset[1]
	ys2 := bg.scaledelta[1] * (pos[1] - zoomybound) * bg.delta[1] * bgscl
	ys := ((100-(pos[1]-zoomybound)*bg.yscaledelta)*bgscl/bg.yscalestart)*bg.scalestart[1] + ys2
	xs := bg.scaledelta[0] * pos[0] * bg.delta[0] * bgscl
	x *= bgscl
	y = y*bgscl + ((float32(sys.gameHeight)-shakeY)/scly-240)/stgscl[1]
	scly *= stgscl[1]
	rect := bg.startrect
	var wscl [2]float32
	for i := range wscl {
		if bg.zoomdelta[i] != math.MaxFloat32 {
			wscl[i] = MaxF(0, scl+(1-scl)*(1-MaxF(0, bg.zoomdelta[i]))) *
				bgscl * lscl[i]
		} else {
			wscl[i] = MaxF(0, scl+(1-scl)*(1-MaxF(0, bg.windowdelta[i]*bgscl))) *
				bgscl * lscl[i]
		}
	}
	startrect0 := (float32(rect[0]) - (pos[0]+bg.camstartx)*bg.windowdelta[0] + (float32(sys.gameWidth)/2/sclx - float32(bg.notmaskwindow)*(float32(sys.gameWidth)/2)*(1/lscl[0]))) * sys.widthScale * wscl[0]
	if !isStage && wscl[0] == 1 {
		startrect0 += float32(sys.gameWidth-320) / 2 * sys.widthScale
	}
	startrect1 := ((float32(rect[1])-pos[1]*bg.windowdelta[1]+(float32(sys.gameHeight)/scly-240))*wscl[1] - shakeY) * sys.heightScale
	rect[0] = int32(math.Floor(float64(startrect0)))
	rect[1] = int32(math.Floor(float64(startrect1)))
	rect[2] = int32(math.Floor(float64(startrect0 + (float32(rect[2]) * sys.widthScale * wscl[0]) - float32(rect[0]))))
	rect[3] = int32(math.Floor(float64(startrect1 + (float32(rect[3]) * sys.heightScale * wscl[1]) - float32(rect[1]))))
	if rect[0] < sys.scrrect[2] && rect[1] < sys.scrrect[3] && rect[0]+rect[2] > 0 && rect[1]+rect[3] > 0 {
		bg.anim.Draw(&rect, x, y, sclx, scly, bg.xscale[0]*bgscl*(bg.scalestart[0]+xs)*xs3, xbs*bgscl*(bg.scalestart[0]+xs)*xs3, ys*ys3,
			xras*x/(AbsF(ys*ys3)*lscl[1]*float32(bg.anim.spr.Size[1])*bg.scalestart[1])*sclx_recip*bg.scalestart[1],
			Rotation{}, float32(sys.gameWidth)/2, bg.palfx, true, 1, false, 1, 0, 0)
	}
}

type bgCtrl struct {
	bg           []*backGround
	currenttime  int32
	starttime    int32
	endtime      int32
	looptime     int32
	_type        BgcType
	x, y         float32
	v            [3]int32
	src          [2]int32
	dst          [2]int32
	add          [3]int32
	mul          [3]int32
	sinadd       [4]int32
	invall       bool
	color        float32
	positionlink bool
	idx          int
	sctrlid      int32
}

func newBgCtrl() *bgCtrl {
	return &bgCtrl{looptime: -1, x: float32(math.NaN()), y: float32(math.NaN())}
}
func (bgc *bgCtrl) read(is IniSection, idx int) {
	bgc.idx = idx
	xy := false
	srcdst := false
	palfx := false
	switch strings.ToLower(is["type"]) {
	case "anim":
		bgc._type = BT_Anim
	case "visible":
		bgc._type = BT_Visible
	case "enable":
		bgc._type = BT_Enable
	case "null":
		bgc._type = BT_Null
	case "palfx":
		bgc._type = BT_PalFX
		palfx = true
		// Default values for PalFX
		bgc.add = [3]int32{0, 0, 0}
		bgc.mul = [3]int32{256, 256, 256}
		bgc.sinadd = [4]int32{0, 0, 0, 0}
		bgc.invall = false
		bgc.color = 1
	case "posset":
		bgc._type = BT_PosSet
		xy = true
	case "posadd":
		bgc._type = BT_PosAdd
		xy = true
	case "remappal":
		bgc._type = BT_RemapPal
		srcdst = true
		// Default values for RemapPal
		bgc.src = [2]int32{-1, 0}
		bgc.dst = [2]int32{-1, 0}
	case "sinx":
		bgc._type = BT_SinX
	case "siny":
		bgc._type = BT_SinY
	case "velset":
		bgc._type = BT_VelSet
		xy = true
	case "veladd":
		bgc._type = BT_VelAdd
		xy = true
	}
	is.ReadI32("time", &bgc.starttime)
	bgc.endtime = bgc.starttime
	is.readI32ForStage("time", &bgc.starttime, &bgc.endtime, &bgc.looptime)
	is.ReadBool("positionlink", &bgc.positionlink)
	if xy {
		is.readF32ForStage("x", &bgc.x)
		is.readF32ForStage("y", &bgc.y)
	} else if srcdst {
		is.readI32ForStage("source", &bgc.src[0], &bgc.src[1])
		is.readI32ForStage("dest", &bgc.dst[0], &bgc.dst[1])
	} else if palfx {
		is.readI32ForStage("add", &bgc.add[0], &bgc.add[1], &bgc.add[2])
		is.readI32ForStage("mul", &bgc.mul[0], &bgc.mul[1], &bgc.mul[2])
		if is.readI32ForStage("sinadd", &bgc.sinadd[0], &bgc.sinadd[1], &bgc.sinadd[2], &bgc.sinadd[3]) {
			if bgc.sinadd[3] < 0 {
				for i := 0; i < 4; i++ {
					bgc.sinadd[i] = -bgc.sinadd[i]
				}
			}
		}
		var tmp int32
		if is.ReadI32("invertall", &tmp) {
			bgc.invall = tmp != 0
		}
		if is.ReadF32("color", &bgc.color) {
			bgc.color = bgc.color / 256
		}
	} else if is.ReadF32("value", &bgc.x) {
		is.readI32ForStage("value", &bgc.v[0], &bgc.v[1], &bgc.v[2])
	}
	is.ReadI32("sctrlid", &bgc.sctrlid)
}
func (bgc *bgCtrl) xEnable() bool {
	return !math.IsNaN(float64(bgc.x))
}
func (bgc *bgCtrl) yEnable() bool {
	return !math.IsNaN(float64(bgc.y))
}

type bgctNode struct {
	bgc      []*bgCtrl
	waitTime int32
}
type bgcTimeLine struct {
	line []bgctNode
	al   []*bgCtrl
}

func (bgct *bgcTimeLine) clear() {
	*bgct = bgcTimeLine{}
}
func (bgct *bgcTimeLine) add(bgc *bgCtrl) {
	if bgc.looptime >= 0 && bgc.endtime > bgc.looptime {
		bgc.endtime = bgc.looptime
	}
	if bgc.starttime < 0 || bgc.starttime > bgc.endtime ||
		bgc.looptime >= 0 && bgc.starttime >= bgc.looptime {
		return
	}
	wtime := int32(0)
	if bgc.currenttime != 0 {
		if bgc.looptime < 0 {
			return
		}
		wtime += bgc.looptime - bgc.currenttime
	}
	wtime += bgc.starttime
	bgc.currenttime = bgc.starttime
	if wtime < 0 {
		bgc.currenttime -= wtime
		wtime = 0
	}
	i := 0
	for ; ; i++ {
		if i == len(bgct.line) {
			bgct.line = append(bgct.line,
				bgctNode{bgc: []*bgCtrl{bgc}, waitTime: wtime})
			return
		}
		if wtime <= bgct.line[i].waitTime {
			break
		}
		wtime -= bgct.line[i].waitTime
	}
	if wtime == bgct.line[i].waitTime {
		bgct.line[i].bgc = append(bgct.line[i].bgc, bgc)
	} else {
		bgct.line[i].waitTime -= wtime
		bgct.line = append(bgct.line, bgctNode{})
		copy(bgct.line[i+1:], bgct.line[i:])
		bgct.line[i] = bgctNode{bgc: []*bgCtrl{bgc}, waitTime: wtime}
	}
}
func (bgct *bgcTimeLine) step(s *Stage) {
	if len(bgct.line) > 0 && bgct.line[0].waitTime <= 0 {
		for _, b := range bgct.line[0].bgc {
			for i, a := range bgct.al {
				if b.idx < a.idx {
					bgct.al = append(bgct.al, nil)
					copy(bgct.al[i+1:], bgct.al[i:])
					bgct.al[i] = b
					b = nil
					break
				}
			}
			if b != nil {
				bgct.al = append(bgct.al, b)
			}
		}
		bgct.line = bgct.line[1:]
	}
	if len(bgct.line) > 0 {
		bgct.line[0].waitTime--
	}
	var el []*bgCtrl
	for i := 0; i < len(bgct.al); {
		s.runBgCtrl(bgct.al[i])
		if bgct.al[i].currenttime > bgct.al[i].endtime {
			el = append(el, bgct.al[i])
			bgct.al = append(bgct.al[:i], bgct.al[i+1:]...)
			continue
		}
		i++
	}
	for _, b := range el {
		bgct.add(b)
	}
}

type stageShadow struct {
	intensity int32
	color     uint32
	yscale    float32
	fadeend   int32
	fadebgn   int32
	xshear    float32
}
type stagePlayer struct {
	startx, starty, startz int32
}
type Stage struct {
	def             string
	bgmusic         string
	name            string
	displayname     string
	author          string
	nameLow         string
	displaynameLow  string
	authorLow       string
	attachedchardef []string
	sff             *Sff
	at              AnimationTable
	bg              []*backGround
	bgc             []bgCtrl
	bgct            bgcTimeLine
	bga             bgAction
	sdw             stageShadow
	p               [2]stagePlayer
	leftbound       float32
	rightbound      float32
	screenleft      int32
	screenright     int32
	zoffsetlink     int32
	reflection      int32
	hires           bool
	resetbg         bool
	debugbg         bool
	bgclearcolor    [3]int32
	localscl        float32
	scale           [2]float32
	bgmvolume       int32
	bgmloopstart    int32
	bgmloopend      int32
	bgmratiolife    int32
	bgmtriggerlife  int32
	bgmtriggeralt   int32
	mainstage       bool
	stageCamera     stageCamera
	stageTime       int32
	constants       map[string]float32
	p1p3dist        float32
	ver             [2]uint16
	reload          bool
	stageprops      StageProps
}

func newStage(def string) *Stage {
	s := &Stage{def: def, leftbound: -1000,
		rightbound: 1000, screenleft: 15, screenright: 15,
		zoffsetlink: -1, resetbg: true, localscl: 1, scale: [...]float32{float32(math.NaN()), float32(math.NaN())},
		bgmratiolife: 30, stageCamera: *newStageCamera(),
		constants: make(map[string]float32), p1p3dist: 25, bgmvolume: 100}
	s.sdw.intensity = 128
	s.sdw.color = 0x808080
	s.sdw.yscale = 0.4
	s.p[0].startx, s.p[1].startx = -70, 70
	s.stageprops = newStageProps()
	return s
}
func loadStage(def string, main bool) (*Stage, error) {
	s := newStage(def)
	str, err := LoadText(def)
	if err != nil {
		return nil, err
	}
	s.sff = &Sff{}
	lines, i := SplitAndTrim(str, "\n"), 0
	s.at = ReadAnimationTable(s.sff, lines, &i)
	i = 0
	defmap := make(map[string][]IniSection)
	for i < len(lines) {
		is, name, _ := ReadIniSection(lines, &i)
		if i := strings.IndexAny(name, " \t"); i >= 0 {
			if name[:i] == "bg" {
				defmap["bg"] = append(defmap["bg"], is)
			}
		} else {
			defmap[name] = append(defmap[name], is)
		}
	}
	if sec := defmap["info"]; len(sec) > 0 {
		var ok bool
		s.name, ok, _ = sec[0].getText("name")
		if !ok {
			s.name = def
		}
		s.displayname, ok, _ = sec[0].getText("displayname")
		if !ok {
			s.displayname = s.name
		}
		s.author, _, _ = sec[0].getText("author")
		s.nameLow = strings.ToLower(s.name)
		s.displaynameLow = strings.ToLower(s.displayname)
		s.authorLow = strings.ToLower(s.author)
		s.ver = [2]uint16{}
		if str, ok := sec[0]["mugenversion"]; ok {
			for k, v := range SplitAndTrim(str, ".") {
				if k >= len(s.ver) {
					break
				}
				if v, err := strconv.ParseUint(v, 10, 16); err == nil {
					s.ver[k] = uint16(v)
				} else {
					break
				}
			}
		}
		// If the MUGEN version is lower than 1.0, use camera pixel rounding (floor)
		if s.ver[0] == 0 {
			s.stageprops.roundpos = true
		}
		if sec[0].LoadFile("attachedchar", []string{def, "", sys.motifDir, "data/"}, func(filename string) error {
			s.attachedchardef = append(s.attachedchardef, filename)
			return nil
		}); err != nil {
			return nil, err
		}
		if main {
			r, _ := regexp.Compile("^round[0-9]+def$")
			for k, v := range sec[0] {
				if r.MatchString(k) {
					re := regexp.MustCompile("[0-9]+")
					submatchall := re.FindAllString(k, -1)
					if len(submatchall) == 1 {
						if err := LoadFile(&v, []string{def, "", sys.motifDir, "data/"}, func(filename string) error {
							if sys.stageList[Atoi(submatchall[0])], err = loadStage(filename, false); err != nil {
								return fmt.Errorf("failed to load %v:\n%v", filename, err)
							}
							return nil
						}); err != nil {
							return nil, err
						}
					}
				}
			}
			sec[0].ReadBool("roundloop", &sys.stageLoop)
		}
	}
	if sec := defmap["constants"]; len(sec) > 0 {
		for key, value := range sec[0] {
			s.constants[key] = float32(Atof(value))
		}
	}
	if sec := defmap["playerinfo"]; len(sec) > 0 {
		sec[0].ReadI32("p1startx", &s.p[0].startx)
		sec[0].ReadI32("p1starty", &s.p[0].starty)
		sec[0].ReadI32("p1startz", &s.p[0].startz)
		sec[0].ReadI32("p2startx", &s.p[1].startx)
		sec[0].ReadI32("p2starty", &s.p[1].starty)
		sec[0].ReadI32("p2startz", &s.p[1].startz)
		sec[0].ReadF32("leftbound", &s.leftbound)
		sec[0].ReadF32("rightbound", &s.rightbound)
		sec[0].ReadF32("p1p3dist", &s.p1p3dist)
	}
	if sec := defmap["scaling"]; len(sec) > 0 {
		if s.ver[0] == 0 { //mugen 1.0+ removed support for topscale
			sec[0].ReadF32("topscale", &s.stageCamera.ztopscale)
		}
	}
	if sec := defmap["bound"]; len(sec) > 0 {
		sec[0].ReadI32("screenleft", &s.screenleft)
		sec[0].ReadI32("screenright", &s.screenright)
	}
	if sec := defmap["stageinfo"]; len(sec) > 0 {
		sec[0].ReadI32("zoffset", &s.stageCamera.zoffset)
		sec[0].ReadI32("zoffsetlink", &s.zoffsetlink)
		sec[0].ReadBool("hires", &s.hires)
		sec[0].ReadBool("resetbg", &s.resetbg)
		sec[0].readI32ForStage("localcoord", &s.stageCamera.localcoord[0],
			&s.stageCamera.localcoord[1])
		sec[0].ReadF32("xscale", &s.scale[0])
		sec[0].ReadF32("yscale", &s.scale[1])
	}
	if math.IsNaN(float64(s.scale[0])) {
		s.scale[0] = 1
	} else if s.hires {
		s.scale[0] *= 2
	}
	if math.IsNaN(float64(s.scale[1])) {
		s.scale[1] = 1
	} else if s.hires {
		s.scale[1] *= 2
	}
	s.localscl = float32(sys.gameWidth) / float32(s.stageCamera.localcoord[0])
	s.stageCamera.localscl = s.localscl
	if sec := defmap["camera"]; len(sec) > 0 {
		sec[0].ReadI32("startx", &s.stageCamera.startx)
		//sec[0].ReadI32("starty", &s.stageCamera.starty) //does nothing in mugen
		sec[0].ReadI32("boundleft", &s.stageCamera.boundleft)
		sec[0].ReadI32("boundright", &s.stageCamera.boundright)
		sec[0].ReadI32("boundhigh", &s.stageCamera.boundhigh)
		sec[0].ReadI32("boundlow", &s.stageCamera.boundlow)
		sec[0].ReadF32("verticalfollow", &s.stageCamera.verticalfollow)
		sec[0].ReadI32("floortension", &s.stageCamera.floortension)
		sec[0].ReadI32("tension", &s.stageCamera.tension)
		sec[0].ReadF32("tensionvel", &s.stageCamera.tensionvel)
		sec[0].ReadI32("overdrawhigh", &s.stageCamera.overdrawhigh) //TODO: not implemented
		sec[0].ReadI32("overdrawlow", &s.stageCamera.overdrawlow)
		sec[0].ReadI32("cuthigh", &s.stageCamera.cuthigh) //TODO: not implemented
		sec[0].ReadI32("cutlow", &s.stageCamera.cutlow)
		sec[0].ReadF32("startzoom", &s.stageCamera.startzoom)
		if sys.cam.ZoomMax == 0 {
			sec[0].ReadF32("zoomin", &s.stageCamera.zoomin)
		} else {
			s.stageCamera.zoomin = sys.cam.ZoomMax
		}
		if sys.cam.ZoomMin == 0 {
			sec[0].ReadF32("zoomout", &s.stageCamera.zoomout)
		} else {
			s.stageCamera.zoomout = sys.cam.ZoomMin
		}
		anchor, _, _ := sec[0].getText("zoomanchor")
		if strings.ToLower(anchor) == "bottom" {
			s.stageCamera.zoomanchor = true
		}
		if sec[0].ReadI32("tensionlow", &s.stageCamera.tensionlow) {
			s.stageCamera.ytensionenable = true
			sec[0].ReadI32("tensionhigh", &s.stageCamera.tensionhigh)
		}
	}
	if sec := defmap["music"]; len(sec) > 0 {
		s.bgmusic = sec[0]["bgmusic"]
		sec[0].ReadI32("bgmvolume", &s.bgmvolume)
		sec[0].ReadI32("bgmloopstart", &s.bgmloopstart)
		sec[0].ReadI32("bgmloopend", &s.bgmloopend)
		sec[0].ReadI32("bgmratio.life", &s.bgmratiolife)
		sec[0].ReadI32("bgmtrigger.life", &s.bgmtriggerlife)
		sec[0].ReadI32("bgmtrigger.alt", &s.bgmtriggeralt)
	}
	if sec := defmap["bgdef"]; len(sec) > 0 {
		if sec[0].LoadFile("spr", []string{def, "", sys.motifDir, "data/"}, func(filename string) error {
			sff, err := loadSff(filename, false)
			if err != nil {
				return err
			}
			*s.sff = *sff
			return nil
		}); err != nil {
			return nil, err
		}
		sec[0].ReadBool("debugbg", &s.debugbg)
		sec[0].readI32ForStage("bgclearcolor", &s.bgclearcolor[0], &s.bgclearcolor[1], &s.bgclearcolor[2])
		sec[0].ReadBool("roundpos", &s.stageprops.roundpos)
	}
	reflect := true
	if sec := defmap["shadow"]; len(sec) > 0 {
		var tmp int32
		if sec[0].ReadI32("intensity", &tmp) {
			s.sdw.intensity = Clamp(tmp, 0, 255)
		}
		var r, g, b int32
		// mugen 1.1 removed support for color
		if (s.ver[0] != 1 || s.ver[1] != 1) && (s.sff.header.Ver0 != 2 || s.sff.header.Ver2 != 1) && sec[0].readI32ForStage("color", &r, &g, &b) {
			r, g, b = Clamp(r, 0, 255), Clamp(g, 0, 255), Clamp(b, 0, 255)
		}
		s.sdw.color = uint32(r<<16 | g<<8 | b)
		sec[0].ReadF32("yscale", &s.sdw.yscale)
		sec[0].ReadBool("reflect", &reflect)
		sec[0].readI32ForStage("fade.range", &s.sdw.fadeend, &s.sdw.fadebgn)
		sec[0].ReadF32("xshear", &s.sdw.xshear)
	}
	if reflect {
		if sec := defmap["reflection"]; len(sec) > 0 {
			var tmp int32
			if sec[0].ReadI32("intensity", &tmp) {
				s.reflection = Clamp(tmp, 0, 255)
			}
		}
	}
	var bglink *backGround
	for _, bgsec := range defmap["bg"] {
		if len(s.bg) > 0 && !s.bg[len(s.bg)-1].positionlink {
			bglink = s.bg[len(s.bg)-1]
		}
		s.bg = append(s.bg, readBackGround(bgsec, bglink,
			s.sff, s.at, float32(s.stageCamera.startx), s.stageprops))
	}
	bgcdef := *newBgCtrl()
	i = 0
	for i < len(lines) {
		is, name, _ := ReadIniSection(lines, &i)
		if len(name) > 0 && name[len(name)-1] == ' ' {
			name = name[:len(name)-1]
		}
		switch name {
		case "bgctrldef":
			bgcdef.bg, bgcdef.looptime = nil, -1
			if ids := is.readI32CsvForStage("ctrlid"); len(ids) > 0 &&
				(len(ids) > 1 || ids[0] != -1) {
				kishutu := make(map[int32]bool)
				for _, id := range ids {
					if kishutu[id] {
						continue
					}
					bgcdef.bg = append(bgcdef.bg, s.getBg(id)...)
					kishutu[id] = true
				}
			} else {
				bgcdef.bg = append(bgcdef.bg, s.bg...)
			}
			is.ReadI32("looptime", &bgcdef.looptime)
		case "bgctrl":
			bgc := newBgCtrl()
			*bgc = bgcdef
			if ids := is.readI32CsvForStage("ctrlid"); len(ids) > 0 {
				bgc.bg = nil
				if len(ids) > 1 || ids[0] != -1 {
					kishutu := make(map[int32]bool)
					for _, id := range ids {
						if kishutu[id] {
							continue
						}
						bgc.bg = append(bgc.bg, s.getBg(id)...)
						kishutu[id] = true
					}
				} else {
					bgc.bg = append(bgc.bg, s.bg...)
				}
			}
			bgc.read(is, len(s.bgc))
			s.bgc = append(s.bgc, *bgc)
		}
	}
	link, zlink := 0, -1
	for i, b := range s.bg {
		if b.positionlink && i > 0 {
			s.bg[i].start[0] += s.bg[link].start[0]
			s.bg[i].start[1] += s.bg[link].start[1]
		} else {
			link = i
		}
		if s.zoffsetlink >= 0 && zlink < 0 && b.id == s.zoffsetlink {
			zlink = i
			s.stageCamera.zoffset += int32(b.start[1] * s.scale[1])
		}
	}
	ratio1 := float32(s.stageCamera.localcoord[0]) / float32(s.stageCamera.localcoord[1])
	ratio2 := float32(sys.gameWidth) / 240
	if ratio1 > ratio2 {
		s.stageCamera.drawOffsetY =
			MinF(float32(s.stageCamera.localcoord[1])*s.localscl*0.5*
				(ratio1/ratio2-1), float32(Max(0, s.stageCamera.overdrawlow)))
	}
	if !s.stageCamera.ytensionenable {
		s.stageCamera.drawOffsetY += MinF(float32(s.stageCamera.boundlow), MaxF(0, float32(s.stageCamera.floortension)*s.stageCamera.verticalfollow)) * s.localscl
	} else {
		s.stageCamera.drawOffsetY += MinF(float32(s.stageCamera.boundlow),
			MaxF(0, (-26+(240/(float32(sys.gameWidth)/float32(s.stageCamera.localcoord[0])))-float32(s.stageCamera.tensionhigh)))) * s.localscl
	}
	//TODO: test if it works reasonably close to mugen
	if sys.gameWidth > s.stageCamera.localcoord[0]*3*320/(s.stageCamera.localcoord[1]*4) {
		if s.stageCamera.cutlow == math.MinInt32 {
			//if omitted, the engine attempts to guess a reasonable set of values
			s.stageCamera.drawOffsetY -= float32(s.stageCamera.localcoord[1]-s.stageCamera.zoffset) / s.localscl //- float32(s.stageCamera.boundlow)*s.localscl
		} else {
			//number of pixels into the bottom of the screen that may be cut from drawing when the screen aspect is shorter than the stage aspect
			if s.stageCamera.cutlow < s.stageCamera.boundlow || s.stageCamera.boundlow <= 0 {
				s.stageCamera.drawOffsetY -= float32(s.stageCamera.cutlow) * s.localscl
			} // else {
			//	s.stageCamera.drawOffsetY -= float32(s.stageCamera.boundlow) * s.localscl
			//}
		}
	}
	s.mainstage = main
	return s, nil
}
func (s *Stage) copyStageVars(src *Stage) {
	s.stageCamera.boundleft = src.stageCamera.boundleft
	s.stageCamera.boundright = src.stageCamera.boundright
	s.stageCamera.boundhigh = src.stageCamera.boundhigh
	s.stageCamera.boundlow = src.stageCamera.boundlow
	s.stageCamera.verticalfollow = src.stageCamera.verticalfollow
	s.stageCamera.floortension = src.stageCamera.floortension
	s.stageCamera.tensionhigh = src.stageCamera.tensionhigh
	s.stageCamera.tensionlow = src.stageCamera.tensionlow
	s.stageCamera.tension = src.stageCamera.tension
	s.stageCamera.startzoom = src.stageCamera.startzoom
	s.stageCamera.zoomout = src.stageCamera.zoomout
	s.stageCamera.zoomin = src.stageCamera.zoomin
	s.stageCamera.ytensionenable = src.stageCamera.ytensionenable
	s.leftbound = src.leftbound
	s.rightbound = src.rightbound
	s.stageCamera.ztopscale = src.stageCamera.ztopscale
	s.screenleft = src.screenleft
	s.screenright = src.screenright
	s.stageCamera.zoffset = src.stageCamera.zoffset
	s.zoffsetlink = src.zoffsetlink
	s.scale[0] = src.scale[0]
	s.scale[1] = src.scale[1]
	s.sdw.intensity = src.sdw.intensity
	s.sdw.color = src.sdw.color
	s.sdw.yscale = src.sdw.yscale
	s.sdw.fadeend = src.sdw.fadeend
	s.sdw.fadebgn = src.sdw.fadebgn
	s.sdw.xshear = src.sdw.xshear
	s.reflection = src.reflection
}
func (s *Stage) getBg(id int32) (bg []*backGround) {
	if id >= 0 {
		for _, b := range s.bg {
			if b.id == id {
				bg = append(bg, b)
			}
		}
	}
	return
}
func (s *Stage) runBgCtrl(bgc *bgCtrl) {
	bgc.currenttime++
	switch bgc._type {
	case BT_Anim:
		a := s.at.get(bgc.v[0])
		if a != nil {
			for i := range bgc.bg {
				masktemp := bgc.bg[i].anim.mask
				srcAlphatemp := bgc.bg[i].anim.srcAlpha
				dstAlphatemp := bgc.bg[i].anim.dstAlpha
				tiletmp := bgc.bg[i].anim.tile
				bgc.bg[i].actionno = bgc.v[0]
				bgc.bg[i].anim = *a
				bgc.bg[i].anim.tile = tiletmp
				bgc.bg[i].anim.dstAlpha = dstAlphatemp
				bgc.bg[i].anim.srcAlpha = srcAlphatemp
				bgc.bg[i].anim.mask = masktemp
			}
		}
	case BT_Visible:
		for i := range bgc.bg {
			bgc.bg[i].visible = bgc.v[0] != 0
		}
	case BT_Enable:
		for i := range bgc.bg {
			bgc.bg[i].visible, bgc.bg[i].active = bgc.v[0] != 0, bgc.v[0] != 0
		}
	case BT_PalFX:
		for i := range bgc.bg {
			bgc.bg[i].palfx.add = bgc.add
			bgc.bg[i].palfx.mul = bgc.mul
			bgc.bg[i].palfx.sinadd[0] = bgc.sinadd[0]
			bgc.bg[i].palfx.sinadd[1] = bgc.sinadd[1]
			bgc.bg[i].palfx.sinadd[2] = bgc.sinadd[2]
			bgc.bg[i].palfx.cycletime = bgc.sinadd[3]
			bgc.bg[i].palfx.invertall = bgc.invall
			bgc.bg[i].palfx.color = bgc.color
		}
	case BT_PosSet:
		for i := range bgc.bg {
			if bgc.xEnable() {
				bgc.bg[i].bga.pos[0] = bgc.x
			}
			if bgc.yEnable() {
				bgc.bg[i].bga.pos[1] = bgc.y
			}
		}
		if bgc.positionlink {
			if bgc.xEnable() {
				s.bga.pos[0] = bgc.x
			}
			if bgc.yEnable() {
				s.bga.pos[1] = bgc.y
			}
		}
	case BT_PosAdd:
		for i := range bgc.bg {
			if bgc.xEnable() {
				bgc.bg[i].bga.pos[0] += bgc.x
			}
			if bgc.yEnable() {
				bgc.bg[i].bga.pos[1] += bgc.y
			}
		}
		if bgc.positionlink {
			if bgc.xEnable() {
				s.bga.pos[0] += bgc.x
			}
			if bgc.yEnable() {
				s.bga.pos[1] += bgc.y
			}
		}
	case BT_RemapPal:
		if bgc.src[0] >= 0 && bgc.src[1] >= 0 && bgc.dst[1] >= 0 {
			// Get source pal
			si, ok := s.sff.palList.PalTable[[...]int16{int16(bgc.src[0]), int16(bgc.src[1])}]
			if !ok || si < 0 {
				return
			}
			var di int
			if bgc.dst[0] < 0 {
				// Set dest pal to source pal (remap gets reset)
				di = si
			} else {
				// Get dest pal
				di, ok = s.sff.palList.PalTable[[...]int16{int16(bgc.dst[0]), int16(bgc.dst[1])}]
				if !ok || di < 0 {
					return
				}
			}
			s.sff.palList.Remap(si, di)
		}
	case BT_SinX, BT_SinY:
		ii := Btoi(bgc._type == BT_SinY)
		if bgc.v[0] == 0 {
			bgc.v[1] = 0
		}
		a := float32(bgc.v[2]) / 360
		st := int32((a - float32(int32(a))) * float32(bgc.v[1]))
		if st < 0 {
			st += Abs(bgc.v[1])
		}
		for i := range bgc.bg {
			bgc.bg[i].bga.radius[ii] = bgc.x
			bgc.bg[i].bga.sinlooptime[ii] = bgc.v[1]
			bgc.bg[i].bga.sintime[ii] = st
		}
		if bgc.positionlink {
			s.bga.radius[ii] = bgc.x
			s.bga.sinlooptime[ii] = bgc.v[1]
			s.bga.sintime[ii] = st
		}
	case BT_VelSet:
		for i := range bgc.bg {
			if bgc.xEnable() {
				bgc.bg[i].bga.vel[0] = bgc.x
			}
			if bgc.yEnable() {
				bgc.bg[i].bga.vel[1] = bgc.y
			}
		}
		if bgc.positionlink {
			if bgc.xEnable() {
				s.bga.vel[0] = bgc.x
			}
			if bgc.yEnable() {
				s.bga.vel[1] = bgc.y
			}
		}
	case BT_VelAdd:
		for i := range bgc.bg {
			if bgc.xEnable() {
				bgc.bg[i].bga.vel[0] += bgc.x
			}
			if bgc.yEnable() {
				bgc.bg[i].bga.vel[1] += bgc.y
			}
		}
		if bgc.positionlink {
			if bgc.xEnable() {
				s.bga.vel[0] += bgc.x
			}
			if bgc.yEnable() {
				s.bga.vel[1] += bgc.y
			}
		}
	}
}
func (s *Stage) action() {
	link, zlink, paused := 0, -1, true
	if sys.tickFrame() && (sys.super <= 0 || !sys.superpausebg) &&
		(sys.pause <= 0 || !sys.pausebg) {
		paused = false
		s.stageTime++
		s.bgct.step(s)
		s.bga.action()
	}
	for i, b := range s.bg {
		b.palfx.step()
		if sys.bgPalFX.enable {
			// TODO: Finish proper synthesization of bgPalFX into PalFX from bg element
			// (Right now, bgPalFX just overrides all unique parameters from BG Elements' PalFX)
			// for j := 0; j < 3; j++ {
			// if sys.bgPalFX.invertall {
			// b.palfx.eAdd[j] = -b.palfx.add[j] * (b.palfx.mul[j]/256) + 256 * (1-(b.palfx.mul[j]/256))
			// b.palfx.eMul[j] = 256
			// }
			// b.palfx.eAdd[j] = int32((float32(b.palfx.eAdd[j])) * sys.bgPalFX.eColor)
			// b.palfx.eMul[j] = int32(float32(b.palfx.eMul[j]) * sys.bgPalFX.eColor + 256*(1-sys.bgPalFX.eColor))
			// }
			// b.palfx.synthesize(sys.bgPalFX)
			b.palfx.eAdd = sys.bgPalFX.eAdd
			b.palfx.eMul = sys.bgPalFX.eMul
			b.palfx.eColor = sys.bgPalFX.eColor
			b.palfx.eInvertall = sys.bgPalFX.eInvertall
			b.palfx.eNegType = sys.bgPalFX.eNegType
		}
		if b.active && !paused {
			s.bg[i].bga.action()
			if i > 0 && b.positionlink {
				s.bg[i].bga.offset[0] += s.bg[link].bga.sinoffset[0]
				s.bg[i].bga.offset[1] += s.bg[link].bga.sinoffset[1]
			} else {
				link = i
			}
			if s.zoffsetlink >= 0 && zlink < 0 && b.id == s.zoffsetlink {
				zlink = i
				s.bga.offset[1] += b.bga.offset[1]
			}
			s.bg[i].anim.Action()
		}
	}
}
func (s *Stage) draw(top bool, x, y, scl float32) {
	bgscl := float32(1)
	if s.hires {
		bgscl = 0.5
	}
	if s.stageCamera.boundhigh > 0 {
		y += float32(s.stageCamera.boundhigh)
	}
	yofs, pos := sys.envShake.getOffset(), [...]float32{x, y}
	scl2 := s.localscl * scl
	if pos[1] <= float32(s.stageCamera.boundlow) && pos[1] < float32(s.stageCamera.boundhigh)-sys.cam.ExtraBoundH {
		yofs += (pos[1]-float32(s.stageCamera.boundhigh))*scl2 +
			sys.cam.ExtraBoundH*scl
		pos[1] = float32(s.stageCamera.boundhigh) - sys.cam.ExtraBoundH/s.localscl
	}
	if s.stageCamera.verticalfollow > 0 {
		if yofs < 0 {
			tmp := (float32(s.stageCamera.boundhigh) - pos[1]) * scl2
			if scl > 1 {
				tmp += (sys.cam.screenZoff + float32(sys.gameHeight-240)) * (1/scl - 1)
			} else {
				tmp += float32(sys.gameHeight) * (1/scl - 1)
			}
			if tmp >= 0 {
			} else if yofs < tmp {
				yofs -= tmp
				pos[1] += tmp / scl2
			} else {
				pos[1] += yofs / scl2
				yofs = 0
			}
		} else {
			if -yofs >= pos[1]*scl2 {
				pos[1] += yofs / scl2
				yofs = 0
			}
		}
	}
	if !sys.cam.ZoomEnable {
		for i, p := range pos {
			pos[i] = float32(math.Ceil(float64(p - 0.5)))
		}
	}
	yofs3 := (s.stageCamera.drawOffsetY +
		float32(s.stageCamera.localcoord[1]-240)*s.localscl)
	yofs4 := ((360*float32(s.stageCamera.localcoord[0]) +
		160*float32(s.stageCamera.localcoord[1])) /
		float32(s.stageCamera.localcoord[0])) / 480
	for _, b := range s.bg {
		if b.visible && b.toplayer == top && b.anim.spr != nil {
			b.draw(pos, scl, bgscl, s.localscl, s.scale,
				yofs+yofs3*Pow(Pow(scl, b.zoomdelta[1]), yofs4)-s.stageCamera.drawOffsetY*(1-b.delta[1]*bgscl), true)
		}
	}
}
func (s *Stage) reset() {
	s.sff.palList.ResetRemap()
	s.bga.clear()
	for i := range s.bg {
		s.bg[i].reset()
	}
	for i := range s.bgc {
		s.bgc[i].currenttime = 0
	}
	s.bgct.clear()
	for i := len(s.bgc) - 1; i >= 0; i-- {
		s.bgct.add(&s.bgc[i])
	}
	s.stageTime = 0
}

func (s *Stage) modifyBGCtrl(id int32, t, v [3]int32, x, y float32, src, dst [2]int32,
	add, mul [3]int32, sinadd [4]int32, invall int32, color float32) {
	for i := range s.bgc {
		if id == s.bgc[i].sctrlid {
			if t[0] != IErr {
				s.bgc[i].starttime = t[0]
			}
			if t[1] != IErr {
				s.bgc[i].endtime = t[1]
			}
			if t[2] != IErr {
				s.bgc[i].looptime = t[2]
			}
			for j := 0; j < 3; j++ {
				if v[j] != IErr {
					s.bgc[i].v[j] = v[j]
				}
			}
			if !math.IsNaN(float64(x)) {
				s.bgc[i].x = x
			}
			if !math.IsNaN(float64(y)) {
				s.bgc[i].y = y
			}
			for j := 0; j < 2; j++ {
				if src[j] != IErr {
					s.bgc[i].src[j] = src[j]
				}
				if dst[j] != IErr {
					s.bgc[i].dst[j] = dst[j]
				}
			}
			var side int32 = 1
			if sinadd[3] != IErr {
				if sinadd[3] < 0 {
					sinadd[3] = -sinadd[3]
					side = -1
				}
			}
			for j := 0; j < 3; j++ {
				if add[j] != IErr {
					s.bgc[i].add[j] = add[j]
				}
				if mul[j] != IErr {
					s.bgc[i].mul[j] = mul[j]
				}
				if sinadd[j] != IErr {
					s.bgc[i].sinadd[j] = sinadd[j] * side
				}
			}
			if invall != IErr {
				s.bgc[i].invall = invall != 0
			}
			if !math.IsNaN(float64(color)) {
				s.bgc[i].color = color / 256
			}
			s.reload = true
		}
	}
}
