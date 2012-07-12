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
	"path/filepath"
	"fmt"
	"log"
)

type SubstackLocation int

const (
	Distal   SubstackLocation = iota
	Proximal SubstackLocation = iota
	Unknown  SubstackLocation = iota
)

const (
	DistalSuperpixels   = 1501268
	DistalSegments      = 774339
	ProximalSuperpixels = 6966702
	ProximalSegments    = 4975511
	Full12kSuperpixels  = 46793902
	Full12kSegments     = 38889751
)

var SubstackDescription = [3]string{
	"Distal",
	"Proximal",
	"Unknown",
}

// GetSubstackLocation returns a SubstackLocation given a string
// description: "Distal", "Proximal", or "12k"
func GetSubstackLocation(location string) SubstackLocation {
	if location == "Distal" {
		return Distal
	} else if location == "Proximal" {
		return Proximal
	} else {
		log.Fatalln("Stack location should be either 'Distal' or 'Proximal'")
	}
	return Unknown
}

const (
	// DistalStackDir was first 161-610 slice TEM data to be proofread
	// and was in the non-seamless space.
	DistalStackDir = "/groups/flyem/proj/data/data_to_be_proofread" +
		"/medulla.HPF.Leginon.3500x.zhiyuan.fall2008" +
		"/region.crop4_global_alignment_0161_1460.unreal.161788539746303_40" +
		"/ms3_1011.1110_4k.4k_09042009" +
		"/shinya.04132010.after_export_07152010_161.610" +
		"/03172011_with.anc.bodies"

	// DistalExportDir is the parent directory of all proofreader exports
	// of assigned synapse tracing for distal, non-seamless stack
	DistalExportDir = "/groups/flyem/proj/data/proofread_data" +
		"/medulla_synapse_driven_proofreading/medulla_0161_0610_anc"

	// SeamlessStackDir is intermediate target stack for all body ID 
	// renumbering in column proofreading.
	SeamlessStackDir = "/groups/flyem/proj/data/data_to_be_proofread" +
		"/medulla.HPF.Leginon.3500x.zhiyuan.fall2008" +
		"/region.crop4_global_alignment_0161_1460.unreal.161788539746303_40" +
		"/ms3_1011.1110_4k.4k_09042009/REF_seamless"

	// SeamlessSynapsesFile is the transformed synapse annotation file
	// from the non-seamless distal space to seamless column space.
	SeamlessSynapsesFile = "/groups/flyem/proj/data/data_to_be_proofread" +
		"/medulla.HPF.Leginon.3500x.zhiyuan.fall2008" +
		"/region.crop4_global_alignment_0161_1460.unreal.161788539746303_40" +
		"/ms3_1011.1110_4k.4k_09042009/REF_seamless" +
		"/annotations-synapses-xformed2.json"

	// SeamlessExportDir is the parent directory of all proofreader exports
	// of assigned synapse tracing for seamless stack
	SeamlessExportDir = "/groups/flyem/proj/data/proofread_data" +
		"/medulla_synapse_driven_proofreading/REF_seamless"

	// Orig12kStackDir is the first 12k x 12k x 1300 stack that should 
	// match body IDs of REF_seamless 5k x 6k stack.
	Orig12kStackDir = "/groups/flyem/data/medulla-TEM-fall2008" +
		"/integrate-20110630/data"

	// Orig12kSynapsesFile is the transformed synapse annotation file
	// for the original 12k x 12k x 1300 stack that has "uid" tags
	// associated with original T-bars before transformation to 12k space
	Orig12kSynapsesFile = "/groups/flyem/data/medulla-TEM-fall2008" +
		"/integrate-20110630/data/annotations-synapses-xformed2.json"
)

// InitialSuperpixelToBodyMapSize returns a guess of the # of superpixels
// for a given stack path.
func InitialSuperpixelToBodyMapSize(path string) int {
	isDistal, _ := filepath.Match(DistalExportDir+"/*", path)
	isProximal, _ := filepath.Match(SeamlessExportDir+"/*", path)
	is12k, _ := filepath.Match("/groups/flyem/data/medulla-TEM-fall2008/*/data",
		path)
	switch {
	case isDistal || path == DistalStackDir:
		return DistalSuperpixels
	case isProximal || path == SeamlessStackDir:
		return ProximalSuperpixels
	case is12k || path == Orig12kStackDir:
		return Full12kSuperpixels
	}
	return DistalSuperpixels // Smallest so we don't overestimate
}

// InitialSegmentToBodyMapSize returns a guess of the # of segments
// for a given stack path.
func InitialSegmentToBodyMapSize(path string) int {
	isDistal, _ := filepath.Match(DistalExportDir+"/*", path)
	isProximal, _ := filepath.Match(SeamlessExportDir+"/*", path)
	is12k, _ := filepath.Match("/groups/flyem/data/medulla-TEM-fall2008/*/data",
		path)
	switch {
	case isDistal || path == DistalStackDir:
		return DistalSegments
	case isProximal || path == SeamlessStackDir:
		return ProximalSegments
	case is12k || path == Orig12kStackDir:
		return Full12kSegments
	}
	return DistalSegments // Smallest so we don't overestimate
}

// ProofreaderUserids is a slice of userids for proofreaders.
var ProofreaderUserids = []string{"abeln", "changl", "lauchies",
	"ogundeyio", "saundersm", "shapirov", "sigmundc", "takemurasa"}

type AssignmentMapping map[string]struct {
	Last int
	Use  []int
}

// ProofreadingExports describes which export sets contain which
// proofreading assignment sets.
var proofreadingExports = [2]AssignmentMapping{
	{
		"abeln":      {4, []int{}},
		"changl":     {5, []int{}},
		"lauchies":   {5, []int{}},
		"ogundeyio":  {5, []int{}},
		"saundersm":  {5, []int{}},
		"shapirov":   {5, []int{}},
		"sigmundc":   {5, []int{}},
		"takemurasa": {5, []int{1}},
	},
	{
		"abeln":     {49, []int{14, 15, 16}},
		"changl":    {49, []int{}},
		"lauchies":  {30, []int{}},
		"ogundeyio": {49, []int{}},
		"saundersm": {49, []int{}},
		"shapirov": {49, []int{1, 2, 3, 5, 8, 9, 13, 14, 15, 16,
			17, 18, 28, 29, 31, 32, 33, 34, 37, 38, 39, 40, 41,
			42, 45, 46, 47, 48}},
		"sigmundc":   {48, []int{1, 2, 6, 8, 9}},
		"takemurasa": {48, []int{1, 2, 3, 4, 5, 6}},
	},
}

// NumAssignmentSets returns the last assignment set done by
// a given proofreader for a substack location
func LastAssignmentSet(userid string, s SubstackLocation) (lastSet int) {
	return proofreadingExports[s][userid].Last
}

// UseAssignmentSet returns the export set number to use when analyzing
// proofreading assignment 'assignedSet'.  The mapping is required since
// some exports are cumulative and others are copied in an ad-hoc fashion.
func UseAssignmentSet(location SubstackLocation, userid string,
	assignedSet int) (setnum int) {

	for i := range proofreadingExports[location][userid].Use {
		if proofreadingExports[location][userid].Use[i] == assignedSet {
			setnum = assignedSet
			return
		}
	}
	setnum = proofreadingExports[location][userid].Last
	return
}

// BaseStackDir returns the directory of the base stack for
// a given substack location.
func BaseStackDir(location SubstackLocation) (dir string) {
	switch location {
	case Distal:
		dir = DistalStackDir
	case Proximal:
		dir = SeamlessStackDir
	default:
		log.Fatalln("FATAL ERROR: Unknown substack", location,
			"in BaseStackDir()")
	}
	return
}

// AssignmentExportDir returns the directory where a given user
// exported a given synapse assignment set.  Note that due to accumulation
// and starting new sessions, exports might cover an abitrary list of
// assignments.
func AssignmentExportDir(location SubstackLocation, userid string,
	setnum int) (dir string) {

	dir = fmt.Sprintf("%s.synapse%d", userid, setnum)
	switch location {
	case Distal:
		dir = filepath.Join(DistalExportDir, dir)
	case Proximal:
		dir = filepath.Join(SeamlessExportDir, dir)
	default:
		log.Fatalln("FATAL ERROR: Unknown substack", location,
			"in AssignmentExportDir()")
	}
	return
}

// AssignmentJsonFilename returns the assignment JSON filename for a
// synapse-driven proofreading assignment.
func AssignmentJsonFilename(location SubstackLocation, userid string,
	setnum int) (filename string) {

	filename = fmt.Sprintf(
		"proofreader_assignments_%d/assigned-synapses-%s.json",
		setnum, userid)
	switch location {
	case Distal:
		filename = filepath.Join(DistalStackDir, filename)
	case Proximal:
		filename = filepath.Join(SeamlessStackDir, filename)
	default:
		log.Fatalln("FATAL ERROR: Unknown substack", location,
			"in AssignmentJsonFilename()")
	}
	return
}
