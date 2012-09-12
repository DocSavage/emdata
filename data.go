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
	"reflect"
	"strconv"
	"time"
)

// MaxCoord returns the maximum of two VoxelCoord
func MaxCoord(i, j VoxelCoord) VoxelCoord {
	if i >= j {
		return i
	}
	return j
}

// MinCoord returns the minimum of two VoxelCoord
func MinCoord(i, j VoxelCoord) VoxelCoord {
	if i <= j {
		return i
	}
	return j
}

// BodyId holds a label for a body.  The 0 body is reserved for
// edges although it is generally deprecated in recent EM segmentation.
// This is a signed quantity because 64-bits provides more than 
// enough headroom for unique bodies, and we may want to represent
// non-unique conditions using the same value, e.g., orphan or leaves.
type BodyId int64

// BodySet is a set of body IDs.
type BodySet map[BodyId]bool

// VoxelCoord holds a coordinate for a voxel.
type VoxelCoord int

// SetWithString sets a VoxelCoord with a number encoded as a string.
func (v *VoxelCoord) SetWithString(s string) error {
	value, err := strconv.Atoi(s)
	*v = VoxelCoord(value)
	return err
}

// String returns the unicode representation.
func (v VoxelCoord) String() string {
	return strconv.Itoa(int(v))
}

// Point2d has X,Y coordinates with axes increasing right then down.
type Point2d [2]VoxelCoord

// X returns the X voxel coordinate
func (pt Point2d) X() VoxelCoord {
	return pt[0]
}

// Y returns the Y voxel coordinate
func (pt Point2d) Y() VoxelCoord {
	return pt[1]
}

// IntX returns integer X voxel coordinate
func (pt Point2d) IntX() int {
	return int(pt[0])
}

// IntY returns integer Y voxel coordinate
func (pt Point2d) IntY() int {
	return int(pt[1])
}

// SqrDistance returns the squared distance between two points
func (pt Point2d) SqrDistance(pt2 Point2d) int {
	dx := int(pt[0] - pt2[0])
	dy := int(pt[1] - pt2[1])
	return dx*dx + dy*dy
}

// String returns representation like "(1,2)"
func (pt Point2d) String() string {
	return "(" + pt[0].String() + "," + pt[1].String() + ")"
}

// PixelsAtRadius returns an array of pixels at a given radius
// within the XY plane.
func (p Point2d) PixelsAtRadius(radius, maxX, maxY int) (pixels []Point2d) {
	if radius == 0 {
		pixels = []Point2d{p}
		return
	}
	r := VoxelCoord(radius)
	x := p.X()
	y := p.Y()
	pixels = make([]Point2d, r*8)
	minXCoord := MaxCoord(0, x-r)
	maxXCoord := MinCoord(VoxelCoord(maxX), x+r)
	minYCoord := MaxCoord(0, y-r)
	maxYCoord := MinCoord(VoxelCoord(maxY), y+r)

	// Check top line
	if y-r >= 0 {
		for ix := minXCoord; ix <= maxXCoord; ix++ {
			pixels = append(pixels, Point2d{ix, y - r})
		}
	}
	// Check bottom line
	if y+r <= maxYCoord {
		for ix := minXCoord; ix <= maxXCoord; ix++ {
			pixels = append(pixels, Point2d{ix, y + r})
		}
	}
	// Check left line
	if x-r >= 0 {
		for iy := minYCoord; iy <= maxYCoord; iy++ {
			pixels = append(pixels, Point2d{x - r, iy})
		}
	}
	// Check right line
	if x+r <= maxXCoord {
		for iy := minYCoord; iy <= maxYCoord; iy++ {
			pixels = append(pixels, Point2d{x + r, iy})
		}
	}
	return
}

// LocationToBodyMap holds 3d Point -> Body Id mappings
type LocationToBodyMap map[Point3d]BodyId

// A Point3d is X,Y,Z coordinate with axes increasing right, down
// and with slices
type Point3d [3]VoxelCoord

// X returns the X voxel coordinate
func (pt Point3d) X() VoxelCoord {
	return pt[0]
}

// Y returns the Y voxel coordinate
func (pt Point3d) Y() VoxelCoord {
	return pt[1]
}

// Z returns the Z voxel coordinate
func (pt Point3d) Z() VoxelCoord {
	return pt[2]
}

// XYZ returns X, Y, and Z coordinates
func (pt Point3d) XYZ() (VoxelCoord, VoxelCoord, VoxelCoord) {
	return pt[0], pt[1], pt[2]
}

// IntX returns integer X voxel coordinate
func (pt Point3d) IntX() int {
	return int(pt[0])
}

// IntY returns integer Y voxel coordinate
func (pt Point3d) IntY() int {
	return int(pt[1])
}

// IntZ returns integer Z voxel coordinate
func (pt Point3d) IntZ() int {
	return int(pt[2])
}

// XYZ returns X, Y, and Z coordinates
func (pt Point3d) IntXYZ() (int, int, int) {
	return int(pt[0]), int(pt[1]), int(pt[2])
}

// Add accumulates the passed point coordinates into current point.
func (pt *Point3d) Add(pt2 Point3d) {
	pt[0] += pt2[0]
	pt[1] += pt2[1]
	pt[2] += pt2[2]
}

// SqrDistance returns the squared distance between two points
func (pt Point3d) SqrDistance(pt2 Point3d) int {
	dx := int(pt[0] - pt2[0])
	dy := int(pt[1] - pt2[1])
	dz := int(pt[2] - pt2[2])
	return dx*dx + dy*dy + dz*dz
}

// String returns representation like "(1,2,3)"
func (pt Point3d) String() string {
	return "(" + pt[0].String() + "," + pt[1].String() + "," +
		pt[2].String() + ")"
}

// Bounds3d defines a bounding box in 3d using MinPt and MaxPt Point3d
type Bounds3d struct {
	MinPt Point3d
	MaxPt Point3d
}

// String returns "(x0,y0,z0) (x1,y1,z1)" bounding box
func (bounds Bounds3d) String() string {
	return bounds.MinPt.String() + " " + bounds.MaxPt.String()
}

// Include returns true if given point is within bounds
func (bounds Bounds3d) Include(pt Point3d) bool {
	if bounds.MinPt[0] > pt[0] || bounds.MaxPt[0] < pt[0] {
		return false
	}
	if bounds.MinPt[1] > pt[1] || bounds.MaxPt[1] < pt[1] {
		return false
	}
	if bounds.MinPt[2] > pt[2] || bounds.MaxPt[2] < pt[2] {
		return false
	}
	return true
}

type cacheData struct {
	data     interface{}
	accessed time.Time
}

type cacheList struct {
	varType  string
	maxItems int
	dataMap  map[string]cacheData
}

// Cache creates a cache for the given type and maximum cache size.
func Cache(cacheType interface{}, maxSize int) (cache cacheList) {
	cache.varType = reflect.TypeOf(cacheType).String()
	cache.maxItems = maxSize
	cache.dataMap = make(map[string]cacheData, maxSize)
	return
}

// Store inserts a data with given key into the cache.  If the maximum
// size of the cache (set during initial Cache() call) is exceeded,
// the oldest item is replaced.
func (cache *cacheList) Store(key string, data interface{}) {
	if len(cache.dataMap) >= cache.maxItems {
		var oldestKey string
		var oldestTime time.Time
		// Remove the last used data item
		itemNum := 0
		for cacheKey, cacheValue := range cache.dataMap {
			if itemNum == 0 || cacheValue.accessed.Before(oldestTime) {
				oldestKey = cacheKey
				oldestTime = cacheValue.accessed
			}
			itemNum++
		}
		delete(cache.dataMap, oldestKey)
	}
	var dataToCache cacheData
	dataToCache.data = data
	dataToCache.accessed = time.Now()
	cache.dataMap[key] = dataToCache
}

// Retrieve fetches the cached data with the given key
func (cache *cacheList) Retrieve(key string) (data interface{}, found bool) {
	cachedObj, found := cache.dataMap[key]
	if found {
		data = cachedObj.data
		cachedObj.accessed = time.Now()
		cache.dataMap[key] = cachedObj
	}
	return
}
