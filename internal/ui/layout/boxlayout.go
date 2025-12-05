package layout

// Dimensions represents the position and size of a box
type Dimensions struct {
	X0, Y0 int // top-left
	X1, Y1 int // bottom-right
}

// Width returns the width of the box
func (d Dimensions) Width() int {
	return d.X1 - d.X0 + 1
}

// Height returns the height of the box
func (d Dimensions) Height() int {
	return d.Y1 - d.Y0 + 1
}

// Direction determines how children are laid out
type Direction int

const (
	ROW    Direction = iota // stack vertically
	COLUMN                  // stack horizontally
)

// Box represents a layout box that can contain other boxes or a window
type Box struct {
	Direction Direction
	Children  []*Box
	Window    string // window name if this is a leaf node
	Size      int    // static size (height for ROW, width for COLUMN)
	Weight    int    // dynamic size weight
}

// ArrangeWindows calculates dimensions for all windows in the layout
func ArrangeWindows(root *Box, x0, y0, width, height int) map[string]Dimensions {
	if len(root.Children) == 0 {
		// leaf node
		if root.Window != "" {
			return map[string]Dimensions{
				root.Window: {X0: x0, Y0: y0, X1: x0 + width - 1, Y1: y0 + height - 1},
			}
		}
		return map[string]Dimensions{}
	}

	var availableSize int
	if root.Direction == COLUMN {
		availableSize = width
	} else {
		availableSize = height
	}

	sizes := calcSizes(root.Children, availableSize)

	result := map[string]Dimensions{}
	offset := 0
	for i, child := range root.Children {
		boxSize := sizes[i]

		var childResult map[string]Dimensions
		if root.Direction == COLUMN {
			childResult = ArrangeWindows(child, x0+offset, y0, boxSize, height)
		} else {
			childResult = ArrangeWindows(child, x0, y0+offset, width, boxSize)
		}

		for k, v := range childResult {
			result[k] = v
		}
		offset += boxSize
	}

	return result
}

func calcSizes(boxes []*Box, availableSpace int) []int {
	totalWeight := 0
	reservedSpace := 0

	for _, box := range boxes {
		if box.Size > 0 {
			reservedSpace += box.Size
		} else {
			weight := box.Weight
			if weight == 0 {
				weight = 1 // default weight
			}
			totalWeight += weight
		}
	}

	dynamicSpace := availableSpace - reservedSpace
	if dynamicSpace < 0 {
		dynamicSpace = 0
	}

	result := make([]int, len(boxes))
	for i, box := range boxes {
		if box.Size > 0 {
			result[i] = min(availableSpace, box.Size)
		} else {
			weight := box.Weight
			if weight == 0 {
				weight = 1
			}
			if totalWeight > 0 {
				result[i] = (dynamicSpace * weight) / totalWeight
			}
		}
	}

	// distribute remainder
	allocated := 0
	for _, s := range result {
		allocated += s
	}
	remainder := availableSpace - allocated
	for i := 0; remainder > 0 && i < len(result); i++ {
		if boxes[i].Size == 0 { // only add to dynamic boxes
			result[i]++
			remainder--
		}
	}

	return result
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
