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
	"log"
	"os"
	"path/filepath"
	"reflect"

	"image"
	_ "image/png"
)

const TileSize = 1024

// ReadSuperpixelTile reads a superpixel tile, either from current
// stack directory or a base stack if necessary.
func ReadSuperpixelTile(stack TiledJsonStack, relTilePath string) (
	superpixels SuperpixelImage, format string, filename string) {

	// Search for file
	filename = filepath.Join(stack.String(), relTilePath)
	_, err := os.Stat(filename)
	if err != nil {
		switch stack.(type) {
		case *BaseStack:
			log.Fatalln("FATAL ERROR: Could not find superpixel tile (",
				relTilePath, ") in base stack (", stack.String(), ")!")
		case *ExportedStack:
			var exported *ExportedStack = stack.(*ExportedStack)
			filename = filepath.Join(exported.Base.String(), relTilePath)
			_, err = os.Stat(filename)
			if err != nil {
				log.Fatalln("FATAL ERROR: Could not find superpixel tile (",
					relTilePath, ") in stack (", exported.String(),
					") or its base (", exported.Base.String(), ")!")
			}
		default:
			log.Fatalln("FATAL ERROR: Bad stack type passed into",
				" ReadSuperpixel Tile:", reflect.TypeOf(stack))
		}
	}

	// Given correct filename, load the image depending on format
	file, err := os.Open(filename)
	if err != nil {
		log.Fatal("FATAL ERROR: opening ", filename, ": ", err)
	}
	defer file.Close()

	superpixels, format, err = image.Decode(file)
	if err != nil {
		log.Fatal("FATAL ERROR: decoding ", filename, ": ", err)
	}
	return
}

type TiledJsonStack interface {
	TilesMetadata() (Bounds3d, SuperpixelFormat)
	JsonStack
	MappedStack
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

	bounds, format := stack.TilesMetadata()
	if !bounds.Include(pt) {
		log.Fatalf("FATAL ERROR: PSD falls outside stack: %s > %s",
			pt, bounds)
	}

	// Compute which tile this point falls within
	x := int(pt.X())
	y := int(pt.Y())
	col := x / TileSize
	row := y / TileSize

	relTilePath := TileFilename(row, col, pt.Z())
	superpixels, _, tilename := ReadSuperpixelTile(stack, relTilePath)

	// Determine relative point within this tile
	tileX := int(pt.X()) - col*TileSize
	tileY := superpixels.Bounds().Max.Y - (int(pt.Y()) - row*TileSize) - 1

	// Get the body id
	var superpixel Superpixel
	superpixel.Slice = uint32(pt.Z())
	superpixel.Label = GetSuperpixelId(superpixels, tileX, tileY, format)

	if superpixel.Label == 0 {
		log.Println("** Warning: PSD falls in ZERO SUPERPIXEL: ", pt)
		log.Println("  Tile:", tilename)
		return BodyId(0)
	}
	bodyId := stack.SuperpixelToBody(superpixel)
	return bodyId
}
