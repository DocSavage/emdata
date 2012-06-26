// Copyright 2012 HHMI.  All rights reserved.
//
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
//     * Redistributions of source code must retain the above copyright
// notice, this list of conditions and the following disclaimer.
//     * Redistributions in binary form must reproduce the above
// copyright notice, this list of conditions and the following disclaimer
// in the documentation and/or other materials provided with the
// distribution.
//     * Neither the name of HHMI nor the names of its
// contributors may be used to endorse or promote products derived from
// this software without specific prior written permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS
// "AS IS" AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT
// LIMITED TO, THE IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR
// A PARTICULAR PURPOSE ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT
// OWNER OR CONTRIBUTORS BE LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL,
// SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT
// LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES; LOSS OF USE,
// DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND ON ANY
// THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE
// OF THIS SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.
//
// Author: katzw@janelia.hhmi.org (Bill Katz)
//  Written as part of the FlyEM Project at Janelia Farm Research Center.

package emdata

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

//	"regexp"
)

const (
	SuperpixelToSegmentFilename = "superpixel_to_segment_map.txt"
	SegmentToBodyFilename       = "segment_to_body_map.txt"
	SuperpixelBoundsFilename    = "superpixel_bounds.txt"
)

// Superpixel is a Raveler-oriented description of a superpixel that
// breaks a unique superpixel id into two components: a slice and a
// unique label within that slice.
type Superpixel struct {
	Slice uint32
	Label uint32
}

// SuperpixelBound holds the top left 2d coord, width, height, 
// and volume (# voxels)
type SuperpixelBound struct {
	MinX   int
	MinY   int
	Width  int
	Height int
	Volume int
}

// Superpixels is a slice of Superpixel type
type Superpixels []Superpixel

// SuperpixelBoundMap maps a superpixel to its bounds
type SuperpixelBoundsMap map[Superpixel]SuperpixelBound

// ReadSuperpixelBounds loads a superpixel bounds file and limits
// returned superpixels to those in the passed-in superpixelSet.
// If superpixelSet is empty, then all superpixels are returned.
func ReadSuperpixelBounds(filename string, superpixelSet map[Superpixel]bool) (
	spBoundsMap SuperpixelBoundsMap, err error) {

	fmt.Println("Loading superpixel bounds:\n", filename)

	file, err := os.Open(filename)
	if err != nil {
		log.Printf("Could not open superpixel bounds: %s\n", filename)
		return
	}
	spBoundsMap = make(SuperpixelBoundsMap)
	linenum := 0
	lineReader := bufio.NewReader(file)
	alwaysSetSuperpixel := len(superpixelSet) == 0
	for {
		line, err := lineReader.ReadString('\n')
		if err != nil {
			break
		}
		linenum++
		if line[0] == ' ' || line[0] == '#' || line[0] == '\n' {
			continue
		}
		var superpixel Superpixel
		var bounds SuperpixelBound
		_, err = fmt.Sscanf(line, "%d %d %d %d %d %d %d",
			&superpixel.Slice, &superpixel.Label,
			&bounds.MinX, &bounds.MinY, &bounds.Width, &bounds.Height,
			&bounds.Volume)
		if err != nil {
			log.Printf("ERROR!!! Cannot parse line %d in %s: %s",
				linenum, filename, err)
		}
		if alwaysSetSuperpixel || superpixelSet[superpixel] {
			spBoundsMap[superpixel] = bounds
		}
	}
	return
}

// SuperpixelToBodyMap holds Superpixel -> Body Id mappings
type SuperpixelToBodyMap map[Superpixel]BodyId

// BodyToSuperpixelMap holds Body Id -> Superpixel mappings
type BodyToSuperpixelsMap map[BodyId]Superpixels

// SuperpixelFormat notes whether superpixel ids, if present, 
// are in 16-bit or 24-bit values.
type SuperpixelFormat uint8

// Enumerate the types of superpixel id formats
const (
	SuperpixelNone SuperpixelFormat = iota
	Superpixel16Bits
	Superpixel24Bits
)

// SuperpixelMapping holds both forward and reverse superpixel<->body maps.
type SuperpixelMapping struct {
	mapLoaded       bool
	spToBodyMap     SuperpixelToBodyMap
	reverseComputed bool
	bodyToSpMap     BodyToSuperpixelsMap
	boundsLoaded    bool
	spBoundsMap     SuperpixelBoundsMap
}

// Stack is a directory that has a base set of capabilities
// shared by all types of stacks (base, session, exported, etc)
type Stack struct {
	Directory string
	SuperpixelMapping
}

// String returns the path of this stack
func (stack Stack) String() string {
	return stack.Directory
}

// StackSuperpixelBoundsFilename returns the file name of the
// synapse annotation file for a given stack
func (stack Stack) StackSuperpixelBoundsFilename() string {
	return filepath.Join(stack.String(), SuperpixelBoundsFilename)
}

// ReadSuperpixelBounds sets a stack's superpixel bounds based on
// the superpixel bounds file in the stack's directory.
func (stack *Stack) ReadSuperpixelBounds() {
	if !stack.boundsLoaded {
		emptySet := map[Superpixel]bool{}
		var err error
		stack.spBoundsMap, err = ReadSuperpixelBounds(
			stack.StackSuperpixelBoundsFilename(), emptySet)
		if err == nil {
			stack.boundsLoaded = true
		}
	}
}

// ReadTxtMaps loads superpixel->body maps and computes reverse
// body->superpixel maps.
func (stack *Stack) ReadTxtMaps(computeReverse bool) {

	if !stack.mapLoaded {
		fmt.Println("Loading superpixel->segment->body maps for stack:\n",
			stack.String())
		stack.spToBodyMap = make(SuperpixelToBodyMap)

		// Load superpixel to segment map
		filename := filepath.Join(stack.String(), SuperpixelToSegmentFilename)
		file, err := os.Open(filename)
		if err != nil {
			log.Fatalf("Could not open %s: %s", filename, err)
		}
		linenum := 0
		lineReader := bufio.NewReader(file)
		for {
			line, err := lineReader.ReadString('\n')
			if err != nil {
				break
			}
			if line[0] == ' ' || line[0] == '#' {
				continue
			}
			var superpixel Superpixel
			var segment BodyId
			if _, err := fmt.Sscanf(line, "%d %d %d", &superpixel.Slice,
				&superpixel.Label, &segment); err != nil {
				log.Fatalf("Error line %d in %s", linenum, filename)
			}
			stack.spToBodyMap[superpixel] = segment // First pass store segment
			linenum++
		}

		// Load segment to body map
		segmentToBodyMap := make(map[BodyId]BodyId)
		filename = filepath.Join(stack.String(), SegmentToBodyFilename)
		file, err = os.Open(filename)
		if err != nil {
			log.Fatalf("Could not open %s", filename)
		}
		linenum = 0
		lineReader = bufio.NewReader(file)
		for {
			line, err := lineReader.ReadString('\n')
			if err != nil {
				break
			}
			if line[0] == ' ' || line[0] == '#' {
				continue
			}
			var segment, body BodyId
			if _, err := fmt.Sscanf(line, "%d %d", &segment, &body); err != nil {
				log.Fatalf("Error line %d in %s", linenum, filename)
			}
			segmentToBodyMap[segment] = body
			linenum++
		}

		// Compute superpixel->body map
		for superpixel, segment := range stack.spToBodyMap {
			stack.spToBodyMap[superpixel] = segmentToBodyMap[segment]
		}

		stack.mapLoaded = true
		fmt.Println("- Maps loaded")

		// Compute reverse if needed
		if computeReverse {
			stack.bodyToSpMap = make(BodyToSuperpixelsMap)
			stack.reverseComputed = true
			fmt.Println("- Reverse maps computed")
		}
	}
}

// SuperpixelToBody returns a body id for a given superpixel.
func (stack *Stack) SuperpixelToBody(s Superpixel) BodyId {
	if !stack.mapLoaded {
		stack.ReadTxtMaps(false)
	}
	return stack.spToBodyMap[s]
}

// BodySuperpixels returns a body->superpixel map for a set of bodies.
func (stack *Stack) BodySuperpixels(useBody map[BodyId]bool) (
	bodyToSpMap BodyToSuperpixelsMap) {

	if !stack.mapLoaded {
		stack.ReadTxtMaps(false)
	}
	bodyToSpMap = make(BodyToSuperpixelsMap)
	for superpixel, bodyId := range stack.spToBodyMap {
		if useBody[bodyId] {
			bodyToSpMap[bodyId] = append(bodyToSpMap[bodyId], superpixel)
		}
	}
	return bodyToSpMap
}

// SuperpixelsChanged looks at the superpixel bounds of two stacks
// for a given set of superpixels and sees if there are any 
// significant changes in the superpixels.
func (stack1 *Stack) SuperpixelsChanged(stack2 *Stack,
	superpixelSet map[Superpixel]bool) bool {

	spBounds1, err1 := ReadSuperpixelBounds(
		stack1.StackSuperpixelBoundsFilename(), superpixelSet)
	if err1 != nil {
		log.Println("** Not able to check if superpixels changed",
			"since superpixel bounds not available for stack:\n", stack1)
		return false
	}
	spBounds2, err2 := ReadSuperpixelBounds(
		stack2.StackSuperpixelBoundsFilename(), superpixelSet)
	if err2 != nil {
		log.Println("** Not able to check if superpixels changed",
			"since superpixel bounds not available for stack:\n", stack2)
		return false
	}

	voxelsTotal := 0
	voxelsDiff := 0
	for superpixel, bounds1 := range spBounds1 {
		voxelsTotal += bounds1.Volume
		bounds2, found := spBounds2[superpixel]
		if !found {
			voxelsDiff += bounds1.Volume
		} else {
			if bounds2.Volume > bounds1.Volume {
				voxelsDiff += bounds2.Volume - bounds1.Volume
			} else {
				voxelsDiff += bounds1.Volume - bounds2.Volume
			}
		}
	}
	percentDiff := float32(voxelsDiff) / float32(voxelsTotal)
	log.Printf("\n%5.2f%% voxel difference in superpixels used to compute overlap analysis between stacks\n", percentDiff)

	if percentDiff > 0.10 {
		log.Fatalln("Error!!  More than 10%% voxel difference in superpixels",
			"between stacks:", percentDiff*100.0, "%% of total",
			voxelsTotal, "voxels\n", stack1, "\n", stack2)
	}
	return false
}

// BaseStackDir is a directory path to a base stack that includes
// all necessary data under one parent directory.
type BaseStack struct {
	Stack
}

func (stack BaseStack) StackSynapsesJsonFilename() string {
	return StackSynapsesJsonFilename(stack.Directory)
}

func (stack BaseStack) StackBodiesJsonFilename() string {
	return StackBodiesJsonFilename(stack.Directory)
}

// TilesMetadata retrieves the 3d bounding box and superpixel format 
// of a stack from the tiles/metadata.txt file.
func (stack BaseStack) TilesMetadata() (Bounds3d, SuperpixelFormat) {

	filename := filepath.Join(stack.Directory, "tiles", "metadata.txt")
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("Could not open tiles/metadata.txt file: %s", filename)
	}
	var bounds Bounds3d
	var superpixelFormat SuperpixelFormat = SuperpixelNone
	minZUnset := true
	maxZUnset := true
	bounds.MinPt[0] = 0
	bounds.MinPt[1] = 0
	lineReader := bufio.NewReader(file)
	for line, err := lineReader.ReadString('\n'); err == nil; line, err = lineReader.ReadString('\n') {

		items := strings.Split(line, "=")
		keyword, value := strings.TrimSpace(items[0]),
			strings.TrimSpace(items[1])
		switch keyword {
		case "width":
			bounds.MaxPt[0].SetWithString(value)
			bounds.MaxPt[0]--
		case "height":
			bounds.MaxPt[1].SetWithString(value)
			bounds.MaxPt[1]--
		case "zmin":
			bounds.MinPt[2].SetWithString(value)
			minZUnset = false
		case "zmax":
			bounds.MaxPt[2].SetWithString(value)
			maxZUnset = false
		case "superpixel-format":
			if value == "RGBA" {
				superpixelFormat = Superpixel24Bits
			} else if value == "I" {
				superpixelFormat = Superpixel16Bits
			} else {
				log.Fatalf("Illegal superpixel-format type (%s): %s",
					value, filename)
			}
		}
	}
	if minZUnset || maxZUnset {
		var errors []string
		if minZUnset {
			errors = append(errors, "zmin not provided")
		}
		if maxZUnset {
			errors = append(errors, "zmax not provided")
		}
		log.Fatalf("Error in reading %s: %s",
			filename, strings.Join(errors, ", "))
	}
	return bounds, superpixelFormat
}

type Overlaps map[BodyId]uint32

type OverlapsMap map[BodyId]Overlaps

// OverlapAnalysis returns a body->body mapping where
// each body in the provided body->superpixel map is matched to
// a body in this stack that has maximal superpixel overlap.
// Quality control is to check if superpixels have changed a lot
// from our target stack using superpixel bounds.
func (stack *BaseStack) OverlapAnalysis(bodyToSpMap BodyToSuperpixelsMap,
	exportedStack *ExportedStack) (bodyToBodyMap map[BodyId]BodyId) {

	stack.ReadTxtMaps(false)
	overlapsMap := make(OverlapsMap)

	superpixelSet := make(map[Superpixel]bool) // Set of used superpixels

	// Go through all superpixels in the provided map and track overlap.
	superpixelsFound := 0
	for bodyId, superpixels := range bodyToSpMap {
		for _, superpixel := range superpixels {
			myBodyId, found := stack.spToBodyMap[superpixel]
			if found {
				superpixelSet[superpixel] = true
				if len(overlapsMap[bodyId]) == 0 {
					overlapsMap[bodyId] = make(Overlaps)
				}
				overlapsMap[bodyId][myBodyId] += 1
				superpixelsFound++
			} else {
				log.Println("Warning!! Superpixel ", superpixel,
					" in traced body is not in target stack (",
					filepath.Base(stack.String()), ")")
			}
		}
	}
	if superpixelsFound != len(superpixelSet) {
		log.Println("\nOverlap analysis: ", superpixelsFound, " of ",
			len(superpixelSet), " superpixels found in target stack (",
			filepath.Base(stack.String()), ")")
	}

	// Quality control: make sure superpixels have not changed a lot
	// from our target stack, else superpixel overlap fails.
	if stack.SuperpixelsChanged(&(exportedStack.Stack), superpixelSet) {
		log.Fatalln("\n*** ERROR: Superpixels changed significantly ",
			"between exported stack (", filepath.Base(exportedStack.String()),
			") and target stack (", filepath.Base(stack.String()), ")")
	}

	bodyToBodyMap = make(map[BodyId]BodyId)
	for bodyId, overlaps := range overlapsMap {
		var largest uint32
		var matchedBodyId BodyId
		for myBodyId, count := range overlaps {
			if count > largest {
				largest = count
				matchedBodyId = myBodyId
			}
		}
		if matchedBodyId == 0 {
			fmt.Println("Warning!! Could not find overlapping body ",
				"for body ", bodyId)
		}
		bodyToBodyMap[bodyId] = matchedBodyId
	}

	return bodyToBodyMap
}

// SessionDir is a directory path to a session, which implies data
// must be also retrieved from its base stack.
type Session struct {
	Stack
	Base BaseStack
}

// ExportedStackDir is a directory path to a legacy exported session
type ExportedStack struct {
	Stack
	Base BaseStack
}

func (stack ExportedStack) StackSynapsesJsonFilename() string {
	return StackSynapsesJsonFilename(stack.Directory)
}

func (stack ExportedStack) StackBodiesJsonFilename() string {
	return StackBodiesJsonFilename(stack.Directory)
}

// TilesMetadata returns tiles metadata from the base stack of
// an exported stack.
func (stack ExportedStack) TilesMetadata() (Bounds3d, SuperpixelFormat) {
	return stack.Base.TilesMetadata()
}
