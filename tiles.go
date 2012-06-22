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
	"fmt"
	"image"
	"image/color"
	"log"
	"os"
	"path/filepath"
	"reflect"

	_ "image/png"
)

const TileSize = 1024

// ReadSuperpixelTile reads a superpixel tile, either from current
// stack directory or a base stack if necessary.
func ReadSuperpixelTile(stack TiledJsonStack, relTilePath string) (
	superpixels image.Image, format string) {

	// Search for file
	filename := filepath.Join(stack.String(), relTilePath)
	_, err := os.Stat(filename)
	if err != nil {
		switch stack.(type) {
		case *BaseStack:
			log.Fatalln("Could not find superpixel tile (", relTilePath,
				") in base stack (", stack.String(), ")!")
		case *ExportedStack:
			var exported *ExportedStack = stack.(*ExportedStack)
			filename = filepath.Join(exported.Base.String(), relTilePath)
			_, err = os.Stat(filename)
			if err != nil {
				log.Fatalln("Could not find superpixel tile (", relTilePath,
					") in stack (", exported.String(), ") or its base (",
					exported.Base.String(), ")!")
			}
		default:
			log.Fatalln("Bad stack type passed into ReadSuperpixel Tile:",
				reflect.TypeOf(stack))
		}
	}

	// Given correct filename, load the image depending on format
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal("Error opening ", filename, ": ", err)
	}
	defer file.Close()

	superpixels, format, err = image.Decode(file)
	if err != nil {
		log.Fatal("Error decoding ", filename, ": ", err)
	}

	return superpixels, format
}

type TiledJsonStack interface {
	String() string
	TilesMetadata() (Bounds3d, SuperpixelFormat)
	JsonStack
	ReadTxtMaps(bool)
	SuperpixelToBody(Superpixel) BodyId
}

// TileFilename returns the path to a given tile relative to a stack root.
func TileFilename(row int, col int, slice VoxelCoord) string {

	var filename string
	if slice >= 1000 {
		sliceDir := (slice / 1000) * 1000
		filename = fmt.Sprintf("tiles/%d/0/%d/%d/s/%d/%d.png", TileSize,
			row, col, sliceDir, slice)
	} else {
		filename = fmt.Sprintf("tiles/%d/0/%d/%d/s/%03d.png", TileSize,
			row, col, slice)
	}
	return filename
}

// GetBodyOfLocation reads the superpixel tile that contains the given point
// in stack space and return its body id.
func GetBodyOfLocation(stack TiledJsonStack, pt Point3d) BodyId {

	bounds, superpixelFormat := stack.TilesMetadata()
	if !bounds.Include(pt) {
		log.Fatalf("PSD falls outside stack boundaries: %s > %s",
			pt, bounds)
	}

	// Read superpixel->body maps if needed.  No reverse maps needed.
	stack.ReadTxtMaps(false)

	// Compute which tile this point falls within
	x := int(pt.X())
	y := int(pt.Y())
	col := x / TileSize
	row := y / TileSize

	relTilePath := TileFilename(row, col, pt.Z())
	superpixels, _ := ReadSuperpixelTile(stack, relTilePath)

	// Determine relative point within this tile
	tileX := int(pt.X()) - col*TileSize
	tileY := superpixels.Bounds().Max.Y - (int(pt.Y()) - row*TileSize) - 1

	// Get the body id
	var superpixel Superpixel
	superpixel.Slice = uint32(pt.Z())

	switch superpixelFormat {
	case Superpixel24Bits:
		r, g, b, _ := superpixels.At(tileX, tileY).RGBA()
		superpixel.Label = uint32((b << 16) | (g << 8) | r)
	case Superpixel16Bits, SuperpixelNone:
		gray16 := superpixels.At(tileX, tileY)
		superpixel.Label = uint32(gray16.(color.Gray16).Y)
	}

	if superpixel.Label == 0 {
		fmt.Println("WARNING: PSD falls in ZERO SUPERPIXEL: ", pt)
		return BodyId(0)
	}
	bodyId := stack.SuperpixelToBody(superpixel)
	return bodyId
}
