package crawl

import (
	"fmt"
	"image"
	"sync"
	"time"
	"xhgm_price_tool/good"
	"xhgm_price_tool/match"

	"github.com/go-vgo/robotgo"
)

func init() {
	robotgo.MouseSleep = 200
}

type CrawlResult struct {
	Total   int
	Valid   int
	Invalid []string
}

func Run(debug bool) (cr CrawlResult, err error) {
	err = InitGamePid()
	if err != nil {
		return
	}
	return run(debug)
}

var x_offset, y_offset int

func run(debug bool) (cr CrawlResult, err error) {
	defer func() {
		if err != nil {
			// logging
		}
	}()

	for sidebarIndex := range 2 {
		var tabs []int
		switch sidebarIndex {
		case 0:
			// tabs = []int{0, 1}
			continue
		case 1:
			tabs = []int{0, 1, 2, 3, 4, 5, 6, 7, 8}
		}
		SwitchSidebar(sidebarIndex)
		var once sync.Once
		for _, tabIndex := range tabs {
			if tabIndex > 5 {
				once.Do(SlideTab)
			}
			SwitchTab(tabIndex)

			rowIndex := -1
			skipCnt := 0
			y_row_offset := 0
			yPad := 0
			for {
				target, err := GetGameScreenshot(GAME_PID, debug)
				if err != nil {
					return cr, err
				}
				itemClips, err := GetItemImages(target, yPad, debug)
				if err != nil {
					return cr, err
				}
				for _, clip := range itemClips {
					if skipCnt > 0 {
						skipCnt--
						continue
					}
					clip.Y += y_row_offset
					key := fmt.Sprintf("%d_%d_%d_%d", sidebarIndex, tabIndex, clip.X, clip.Y)
					item, ok := good.Pos2Item[key]
					if !ok {
						continue
					}
					cr.Total++
					price := match.GetPrice(clip.Image, false)
					fmt.Printf("%s: %d\n", item.Name, price)
					if price > item.PriceRange[1] || price < item.PriceRange[0] {
						cr.Invalid = append(cr.Invalid, item.Name)
						continue
					}
					cr.Valid++
					item.Price = price
				}
				scrollRows, ok := hscroll[fmt.Sprintf("%d_%d", sidebarIndex, tabIndex)]
				if ok {
					rowIndex++
					if rowIndex >= len(scrollRows) {
						break
					} else if rowIndex == len(scrollRows)-1 {
						yPad = 10
					}
					SlideNextPage(scrollRows[rowIndex])
					y_row_offset += scrollRows[rowIndex]
					skipCnt += (3 - scrollRows[rowIndex]) * 6
				} else {
					break
				}
			}
			if rowIndex != -1 {
				SlideToTop()
			}
			time.Sleep(2 * time.Second)
		}
		time.Sleep(2 * time.Second)
	}
	return cr, err
}

var GAME_PID int = -1

func InitGamePid() (err error) {
	ids, err := robotgo.FindIds("Star.exe")
	if err != nil {
		return fmt.Errorf("Game process not found: %w", err)
	}
	if len(ids) == 0 {
		return fmt.Errorf("Game process not found")
	}
	// var w, h int
	// for _, id := range ids {
	// 	x_offset, y_offset, w, h = robotgo.GetBounds(id)
	// 	if w != 1936 && h != 1119 {
	// 		continue
	// 	}
	// 	GAME_PID = id
	// 	break
	// }
	GAME_PID = ids[0]
	x_offset, y_offset, _, _ = robotgo.GetBounds(ids[0])
	// if GAME_PID == -1 {
	// 	return fmt.Errorf("Incorrect resolution, please set resolution to windowed 1920x1080, current resolution %vx%v", w-16, h-29)
	// }
	err = robotgo.ActivePid(ids[0])
	if err != nil {
		return fmt.Errorf("Failed to activate game process: %w", err)
	}
	return nil
}

func GetGameScreenshot(pid int, debug bool) (screenshot image.Image, err error) {
	robotgo.ActivePid(pid)
	screenshot, err = robotgo.Capture(robotgo.GetBounds(pid))
	if err != nil {
		return nil, err
	}
	if debug {
		robotgo.Save(screenshot, fmt.Sprintf("./tmp_%s.png", time.Now().Format(time.DateTime)))
	}
	return screenshot, nil
}
