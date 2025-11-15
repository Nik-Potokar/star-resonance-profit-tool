package crawl

import (
	"fmt"
	"image"

	"github.com/go-vgo/robotgo"
	"github.com/noelyahan/mergi"
)

// Item box
var item_start_box_width = 85
var item_start_box_height = 44
var item_start_box_left = 520
var item_start_box_top = 389
var item_start_box = []int{item_start_box_left, item_start_box_top, item_start_box_left + item_start_box_width, item_start_box_top + item_start_box_height}
var item_horizontal_space = 238
var item_vertical_space = 250

// Left primary category
var sidebar_start_box_width = 240
var sidebar_start_box_height = 80
var sidebar_start_box_left = 30
var sidebar_start_box_top = 360
var sidebar_start_box = []int{sidebar_start_box_left, sidebar_start_box_top, sidebar_start_box_left + sidebar_start_box_width, sidebar_start_box_top + sidebar_start_box_height}
var sidebar_vertical_space = 85

// Top secondary category
var tab_start_box_width = 160
var tab_start_box_height = 60
var tab_start_box_left = 390
var tab_start_box_top = 150
var tab_start_box = []int{tab_start_box_left, tab_start_box_top, tab_start_box_left + tab_start_box_width, tab_start_box_top + tab_start_box_height}
var tab_horizontal_space = 161

type ItemImageClip struct {
	X     int
	Y     int
	Image image.Image
}

func GetItemImages(img image.Image, yPad int, debug bool) (clips []ItemImageClip, err error) {
	for y := range 3 {
		for x := range 6 {
			clip, _ := mergi.Crop(img, image.Pt(item_start_box[0]+x*item_horizontal_space, item_start_box[1]+y*item_vertical_space+yPad), image.Pt(item_start_box_width, item_start_box_height))
			if debug {
				robotgo.Save(clip, fmt.Sprintf("%v_%v.png", x, y))
			}
			clips = append(clips, ItemImageClip{
				X:     x,
				Y:     y,
				Image: clip,
			})
		}
	}
	return
}
