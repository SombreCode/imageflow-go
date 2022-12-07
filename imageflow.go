// Package imageflow is a fast image processing library
package imageflow

import (
	"encoding/json"
)

// Steps is the builder for creating a operation
type Steps struct {
	inputs     []ioOperation
	outputs    []ioOperation
	vertex     []interface{}
	last       uint
	innerGraph graph
	ioID       int
}

// Decode is used to import a image
func (steps *Steps) Decode(task ioOperation) *Steps {
	steps.inputs = append(steps.inputs, task)
	task.setIo(uint(steps.ioID))
	steps.vertex = append(steps.vertex, decode{
		IoID: steps.ioID,
	}.toStep())
	steps.ioID++
	steps.last = uint(len(steps.vertex) - 1)
	return steps
}

// ConstrainWithin is used to constraint a image
func (steps *Steps) ConstrainWithin(w float64, h float64) *Steps {
	steps.input(constrainWithinMap(w, h))
	return steps
}

// ConstrainWithinH is used to constraint a image
func (steps *Steps) ConstrainWithinH(h float64) *Steps {
	steps.input(constrainWithinMap(nil, h))
	return steps
}

// ConstrainWithinW is used to constraint a image
func (steps *Steps) ConstrainWithinW(w float64) *Steps {
	steps.input(constrainWithinMap(w, nil))
	return steps
}

func constrainWithinMap(w interface{}, h interface{}) map[string]interface{} {
	constrainMap := make(map[string]interface{})
	dataMap := make(map[string]interface{})
	dataMap["mode"] = "within"
	if w != nil {
		dataMap["w"] = w
	}
	if h != nil {
		dataMap["h"] = h
	}
	constrainMap["constrain"] = dataMap

	return constrainMap
}

// Constrain is used to constraint a image
func (steps *Steps) Constrain(dataMap Constrain) *Steps {
	steps.input(dataMap.toStep())
	return steps
}

// Encode is used to convert the image
func (steps *Steps) Encode(task ioOperation, preset presetInterface) *Steps {
	task.setIo(uint(steps.ioID))
	steps.outputs = append(steps.outputs, task)
	steps.input(encode{
		IoID:   steps.ioID,
		Preset: preset.toPreset(),
	}.toStep())
	steps.ioID++
	return steps
}

// Rotate90 is to used to rotate by 90 degrees
func (steps *Steps) Rotate90() *Steps {
	rotate := rotate90{}
	steps.input(rotate.toStep())
	return steps
}

// Rotate180 is to used to rotate by 180 degrees
func (steps *Steps) Rotate180() *Steps {
	rotate := rotate180{}
	steps.input(rotate.toStep())
	return steps
}

// Rotate270 is to used to rotate by 270 degrees
func (steps *Steps) Rotate270() *Steps {
	rotate := rotate270{}
	steps.input(rotate.toStep())
	return steps
}

// FlipH is to used to flip image horizontally
func (steps *Steps) FlipH() *Steps {
	rotate := flipH{}
	steps.input(rotate.toStep())
	return steps
}

// FlipV is to used to flip image horizontally
func (steps *Steps) FlipV() *Steps {
	rotate := flipV{}
	steps.input(rotate.toStep())
	return steps
}

func (steps *Steps) input(step interface{}) {
	steps.vertex = append(steps.vertex, step)
	steps.innerGraph.AddEdge(steps.last, uint(len(steps.vertex)-1), "input")
	steps.last = uint(len(steps.vertex) - 1)
}

func (steps *Steps) canvas(f func(*Steps), step stepInterface) *Steps {
	last := steps.last
	f(steps)
	steps.vertex = append(steps.vertex, step.toStep())
	steps.innerGraph.AddEdge(last, uint(len(steps.vertex)-1), "input")
	steps.innerGraph.AddEdge(steps.last, uint(len(steps.vertex)-1), "canvas")
	steps.last = uint(len(steps.vertex) - 1)
	return steps
}

// CopyRectangle copy a image
func (steps *Steps) CopyRectangle(f func(steps *Steps), rect RectangleToCanvas) *Steps {
	return steps.canvas(f, rect)
}

// DrawExact copy a image
func (steps *Steps) DrawExact(f func(steps *Steps), rect DrawExact) *Steps {
	return steps.canvas(f, rect)
}

// Execute the graph
func (steps *Steps) Execute() (map[string][]byte, error) {
	js := steps.ToJSON()
	job := newJob()
	defer job.CleanUp()

	for i := 0; i < len(steps.inputs); i++ {
		data, errorInBuffer := steps.inputs[i].toBuffer()
		if errorInBuffer != nil {
			return nil, errorInBuffer
		}
		errorInInput := job.AddInput(steps.inputs[i].getIo(), data)
		if errorInInput != nil {
			return nil, errorInInput
		}
	}
	for i := 0; i < len(steps.outputs); i++ {
		errorInOutput := job.AddOutput(steps.outputs[i].getIo())
		if errorInOutput != nil {
			return nil, errorInOutput
		}
	}
	errorInMessage := job.Message(js)

	if errorInMessage != nil {
		return nil, errorInMessage
	}

	bufferMap := make(map[string][]byte)
	for i := 0; i < len(steps.outputs); i++ {
		data, errorInOutput := job.GetOutput(steps.outputs[i].getIo())
		if errorInOutput != nil {
			return nil, errorInOutput
		}
		bufferMap = steps.outputs[i].toOutput(data, bufferMap)
	}
	return bufferMap, nil
}

// Branch create a alternate path for the output
func (steps *Steps) Branch(f func(*Steps)) *Steps {
	last := steps.last
	f(steps)
	steps.last = last
	return steps
}

// Region is used to crop or add padding to image
func (steps *Steps) Region(region Region) *Steps {
	steps.input(region.toStep())
	return steps
}

// RegionPercentage is used to crop or add padding to image using percentage
func (steps *Steps) RegionPercentage(region RegionPercentage) *Steps {
	steps.input(region.toStep())
	return steps
}

// CropWhitespace is used to remove whitespace around the image
func (steps *Steps) CropWhitespace(threshold int, padding float64) *Steps {
	steps.input(cropWhitespace{Threshold: threshold, PercentagePadding: padding}.toStep())
	return steps
}

// FillRect is used create a rectangle on the image
func (steps *Steps) FillRect(x1 float64, y1 float64, x2 float64, y2 float64, color Color) *Steps {
	steps.input(fillRect{X1: x1, Y1: y1, X2: x2, Y2: y2, Color: color}.toStep())
	return steps
}

// ExpandCanvas is used create a rectangle on the image
func (steps *Steps) ExpandCanvas(canvas ExpandCanvas) *Steps {
	steps.input(canvas.toStep())
	return steps
}

// Watermark is used to watermark a image
func (steps *Steps) Watermark(data ioOperation, gravity interface{}, fitMode string, fitBox FitBox, opacity float32, hint interface{}) *Steps {
	data.setIo(uint(steps.ioID))
	steps.inputs = append(steps.inputs, data)
	steps.input(watermark{
		IoID:    uint(steps.ioID),
		Gravity: gravity,
		FitMode: fitMode,
		FitBox:  fitBox,
		Opacity: opacity,
		Hints:   hint,
	}.toStep())
	steps.ioID++
	return steps
}

// Command is used to execute query like strings
func (steps *Steps) Command(cmd string) *Steps {
	cmdMap := make(map[string]map[string]string)
	dataMap := make(map[string]string)
	dataMap["kind"] = "ir4"
	dataMap["value"] = cmd
	cmdMap["command_string"] = dataMap
	steps.input(cmdMap)
	return steps
}

// WhiteBalanceSRGB histogram area
// This command is not recommended as it operates in the sRGB space and does not produce perfect results.
func (steps *Steps) WhiteBalanceSRGB(threshold float32) *Steps {
	steps.input(doubleMap("white_balance_histogram_area_threshold_srgb", "threshold", threshold))
	return steps
}

// GrayscaleNTSC histogram area
// This command is not recommended as it operates in the sRGB space and does not produce perfect results.
func (steps *Steps) GrayscaleNTSC() *Steps {
	return steps.colorFilterSRGB("grayscale_ntsc")
}

// GrayscaleFlat histogram area
// This command is not recommended as it operates in the sRGB space and does not produce perfect results.
func (steps *Steps) GrayscaleFlat() *Steps {
	return steps.colorFilterSRGB("grayscale_flat")
}

// GrayscaleBT709 histogram area
// This command is not recommended as it operates in the sRGB space and does not produce perfect results.
func (steps *Steps) GrayscaleBT709() *Steps {
	return steps.colorFilterSRGB("grayscale_bt709")
}

// GrayscaleRY histogram area
// This command is not recommended as it operates in the sRGB space and does not produce perfect results.
func (steps *Steps) GrayscaleRY() *Steps {
	return steps.colorFilterSRGB("grayscale_ry")
}

// Sepia histogram area
// This command is not recommended as it operates in the sRGB space and does not produce perfect results.
func (steps *Steps) Sepia() *Steps {
	return steps.colorFilterSRGB("sepia")
}

// Invert histogram area
// This command is not recommended as it operates in the sRGB space and does not produce perfect results.
func (steps *Steps) Invert() *Steps {
	return steps.colorFilterSRGB("invert")
}

func (steps *Steps) colorFilterSRGB(value string) *Steps {
	steps.input(singleMap("color_filter_srgb", value))
	return steps
}

// Alpha histogram area
// This command is not recommended as it operates in the sRGB space and does not produce perfect results.
func (steps *Steps) Alpha(value float32) *Steps {
	return steps.colorFilterSRGBValue("alpha", value)
}

// Contrast histogram area
// This command is not recommended as it operates in the sRGB space and does not produce perfect results.
func (steps *Steps) Contrast(value float32) *Steps {
	return steps.colorFilterSRGBValue("contrast", value)
}

// Brightness histogram area
// This command is not recommended as it operates in the sRGB space and does not produce perfect results.
func (steps *Steps) Brightness(value float32) *Steps {
	return steps.colorFilterSRGBValue("brightness", value)
}

// Saturation histogram area
// This command is not recommended as it operates in the sRGB space and does not produce perfect results.
func (steps *Steps) Saturation(value float32) *Steps {
	return steps.colorFilterSRGBValue("saturation", value)
}

// PNG encodes to a png
func (steps *Steps) PNG(operation ioOperation) *Steps {
	return steps.Encode(operation, LosslessPNG{})

}

// JPEG encodes to a jpeg
func (steps *Steps) JPEG(operation ioOperation) *Steps {
	return steps.Encode(operation, MozJPEG{})

}

// WebP encodes to a webp
func (steps *Steps) WebP(operation ioOperation) *Steps {
	return steps.Encode(operation, WebPLossless{})

}

// GIF encodes to a gif
func (steps *Steps) GIF(operation ioOperation) *Steps {
	return steps.Encode(operation, GIF{})

}

func (steps *Steps) colorFilterSRGBValue(name string, value float32) *Steps {
	steps.input(doubleMap("color_filter_srgb", name, value))
	return steps
}

// Step specify different nodes
type stepInterface interface {
	toStep() interface{}
}

type edge struct {
	Kind string `json:"kind"`
	To   uint   `json:"to"`
	From uint   `json:"from"`
}
type graph struct {
	edges []edge
}

func (gr *graph) AddEdge(from uint, to uint, kind string) {
	gr.edges = append(gr.edges, edge{To: to, Kind: kind, From: from})
}

// NewStep creates a step that can be used to specify how graph should be processed
func NewStep() Steps {
	return Steps{
		vertex: []interface{}{},
		last:   0,
		ioID:   0,
		innerGraph: graph{
			edges: []edge{},
		},
	}
}

func (steps *Steps) ToJSON() []byte {
	nodeMap := make(map[int]interface{})
	for i := 0; i < len(steps.vertex); i++ {
		nodeMap[i] = steps.vertex[i]
	}
	jsonMap := map[string]interface{}{"framewise": map[string]interface{}{
		"graph": map[string]interface{}{"nodes": nodeMap, "edges": steps.innerGraph.edges},
	}}
	js, _ := json.Marshal(jsonMap)
	return js
}
