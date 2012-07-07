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
	"strconv"
)

// TracingResult gives the result of a proofreader tracing a process.
// Expected results are Orphan, Leaves (exits image volume), or
// a body id, presumably of an anchor body.
type TracingResult int64

const (
	Orphan TracingResult = -2
	Leaves TracingResult = -1
	// Any TracingResult >= 0 is Body Id of anchor
)

// String returns "Orphan", "Leaves" or the stringified body ID
func (result TracingResult) String() string {
	if result == Orphan {
		return "Orphan"
	} else if result == Leaves {
		return "Leaves"
	}
	return strconv.FormatInt(int64(result), 10)
}

// TracingAgent is a unique id that describes a proofreading agent.
type TracingAgent string

// CreatePsdTracing creates a PsdTracing struct by examining each assigned
// location and determining the exported body ID of the stack for that location.
func CreatePsdTracing(location SubstackLocation, userid string, setnum int,
	exportedStack ExportedStack) (tracing *JsonSynapses, psdBodies BodySet) {

	// Read in the assignment JSON: set of PSDs
	jsonFilename := AssignmentJsonFilename(location, userid, setnum)
	tracing = ReadSynapsesJson(jsonFilename)
	log.Println("Read assignment Json:", len(tracing.Data), "synapses")

	// Read in the exported body annotations to determine whether PSD was
	// traced to anchor body or it was orphan/leaves.
	bodyToNotesMap := ReadStackBodyAnnotations(exportedStack)
	log.Println("Read exported bodies Json:", len(bodyToNotesMap), "bodies")

	// For each PSD, find body associated with it using superpixel tiles
	// and the exported session's map.
	definitiveAnchor := 0
	commentedAnchor := 0
	commentedOrphan := 0
	presumedLeaves := 0
	noBodyAnnotated := 0
	psdBodies = make(BodySet) // Set of PSD bodies

	synapses := (*tracing).Data
	for s, _ := range synapses {
		synapses[s].Tbar.Assignment = fmt.Sprintf("%s-%d",
			SubstackDescription[location], setnum)
		psds := synapses[s].Psds
		for p, _ := range psds {
			bodyId := GetBodyOfLocation(&exportedStack, psds[p].Location)
			bodyNote, found := bodyToNotesMap[bodyId]
			if found {
				var tracingResult TracingResult
				if len(bodyNote.Anchor) != 0 {
					definitiveAnchor++
					psdBodies[bodyId] = true
					tracingResult = TracingResult(bodyId)
				} else if bodyNote.AnchorComment() {
					commentedAnchor++
					psdBodies[bodyId] = true
					tracingResult = TracingResult(bodyId)
				} else if bodyNote.OrphanComment() {
					commentedOrphan++
					tracingResult = Orphan
				} else {
					presumedLeaves++
					tracingResult = Leaves
				}
				if len(psds[p].Tracings) == 0 {
					psds[p].Tracings = make(map[string]TracingResult)
				}
				psds[p].Tracings[userid] = tracingResult
			} else {
				noBodyAnnotated++
				log.Println("WARNING!!! PSD ", psds[p].Location, " -> ",
					"exported body ", bodyId, " cannot be found in ",
					"body annotation file for exported stack... skipping")
			}
		}
	}

	log.Println("  Anchors marked with anchor tag: ", definitiveAnchor)
	log.Println(" Anchors detected within comment: ", commentedAnchor)
	log.Println(" Orphans detected within comment: ", commentedOrphan)
	log.Println("PSD bodies with no anchor/orphan: ", presumedLeaves)
	if noBodyAnnotated > 0 {
		log.Println("*** PSD bodies not annotated: ", noBodyAnnotated)
	}
	return
}

// TransformBodies applies a body->body map to transform any traced bodies.
func (tracing *JsonSynapses) TransformBodies(bodyToBodyMap map[BodyId]BodyId) (
	psdBodies BodySet) {

	psdBodies = make(BodySet)
	numErrors := 0
	altered := 0
	unaltered := 0
	synapses := (*tracing).Data
	for s, _ := range synapses {
		psds := synapses[s].Psds
		for p, _ := range psds {
			for userid, result := range psds[p].Tracings {
				if result != Orphan && result != Leaves && result != 0 {
					origBody := BodyId(result)
					targetBody, found := bodyToBodyMap[origBody]
					if !found {
						log.Println("ERROR!!! Body->body map does not contain",
							"PSD body ", result)
						numErrors++
					} else if origBody != targetBody {
						psds[p].Tracings[userid] = TracingResult(targetBody)
						psdBodies[targetBody] = true
						altered++
					} else {
						psdBodies[origBody] = true
						unaltered++
					}
				}
			}
		}
	}

	if numErrors > 0 {
		log.Fatalln("Aborting... found ", numErrors,
			" when transforming PSD bodies.")
	}
	log.Printf("Transformed %d of %d PSD bodies\n", altered, altered+unaltered)
	return
}

type PsdSignature struct {
	Body BodyId
	Z    VoxelCoord
}

func (signature PsdSignature) String() string {
	return fmt.Sprintf("{ Body: %d, Z: %d }", signature.Body, signature.Z)
}

// TransformSynapses modifies synapse locations (T-bar and PSDs) based
// on a transformed synapses annotation list with 'uid' tags and PSDs that
// have matching body IDs
func (tracing *JsonSynapses) TransformSynapses(xformed JsonSynapses) {

	// Construct a lookup map based on 'uid' tag that points to synapse #
	// in the xformed list
	uidMap := make(map[string]int)
	for i, synapse := range xformed.Data {
		uidMap[synapse.Tbar.Uid] = i
	}

	// Go through each traced synapse and match it to associated xformed
	// T-bar or PSD.
	numPsdErrors := 0
	numTbarErrors := 0
	alteredPsds := 0
	alteredTbar := 0
	synapses := (*tracing).Data
	for s, _ := range synapses {
		// Alter T-bar location
		var uid string
		if synapses[s].Tbar.Uid == "" {
			x, y, z := synapses[s].Tbar.Location.XYZ()
			uid = fmt.Sprintf("%05d-%05d-%05d", x, y, z)
			synapses[s].Tbar.Uid = uid
		} else {
			uid = synapses[s].Tbar.Uid
		}
		i, found := uidMap[uid]
		if !found {
			numTbarErrors++
			log.Printf("** Could not find uid %s with xformed synapse list!\n",
				uid)
		} else {
			synapses[s].Tbar.Location = xformed.Data[i].Tbar.Location
			alteredTbar++

			// Alter PSD locations using (body ID, Z coord) signature
			// of transformed synapses.  In cases where we can't resolve
			// two PSDs, emit a warning and just choose PSDs in order.
			xformedPsds := xformed.Data[i].Psds
			psdBodies := make(map[PsdSignature]([]int))
			for p, _ := range xformedPsds {
				signature := PsdSignature{xformedPsds[p].Body,
					xformedPsds[p].Location.Z()}
				_, found := psdBodies[signature]
				if found {
					log.Printf("-- Duplicate PSD signature for %s: %s",
						uid, signature)
					psdBodies[signature] = append(psdBodies[signature], p)
				} else {
					psdBodies[signature] = make([]int, 1)
					psdBodies[signature][0] = p
				}
			}
			psds := synapses[s].Psds
			for p, _ := range psds {
				signature := PsdSignature{psds[p].Body, psds[p].Location.Z()}
				_, found := psdBodies[signature]
				if !found {
					numPsdErrors++
					log.Printf("** Found no match for PSD %s (body %d) tbar %s\n",
						psds[p].Location.String(), psds[p].Body, uid)
				} else {
					alteredPsds++
					first := psdBodies[signature][0]
					psds[p].Location = xformedPsds[first].Location
					if len(psdBodies[signature]) > 1 {
						psdBodies[signature] = psdBodies[signature][1:]
					}
				}
			}
		}
	}

	log.Printf("Transformed locations of %d T-bars and %d PSDs\n",
		alteredTbar, alteredPsds)
	if numTbarErrors > 0 || numPsdErrors > 0 {
		log.Fatalln("Aborting...", numTbarErrors, "uids unmatched",
			"and", numPsdErrors, "PSDs unmatched using signatures")
	}
	return
}
