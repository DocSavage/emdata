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
	Orphan    TracingResult = -2
	Leaves    TracingResult = -1
	Edge      TracingResult = 0
	MinAnchor TracingResult = 1
	// Any TracingResult >= Anchor is Body Id of anchor
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
func CreatePsdTracing(stackId StackId, userid string, setnum int,
	exportedStack ExportedStack, baseStack BaseStack) (
	tracing *JsonSynapses, psdBodies BodySet) {

	psdBodies = make(BodySet) // Set of all PSD bodies

	// Make a closure that adds a traced body to a PSD and modifies
	// the psdBodies set.
	addTracedBody := func(psd *JsonPsd, bodyId BodyId, bodyNote JsonBody) (
		pTracing *JsonTracing) {

		tracingResult := bodyNote.GetTracingResult(bodyId)
		if tracingResult > MinAnchor {
			psdBodies[bodyId] = true
		}
		var tracing JsonTracing
		tracing.Userid = userid
		tracing.Result = tracingResult
		tracing.Stack = StackDescription[stackId]
		tracing.AssignmentSet = setnum
		if tracingResult >= MinAnchor {
			tracing.ExportedBody = bodyId
		}
		numTracings := len((*psd).Tracings)
		if numTracings == 0 {
			(*psd).Tracings = []JsonTracing{tracing}
			pTracing = &((*psd).Tracings[0])
		} else {
			(*psd).Tracings = append((*psd).Tracings, tracing)
			pTracing = &((*psd).Tracings[numTracings])
		}
		return pTracing
	}

	// Read in the assignment JSON: set of PSDs
	jsonFilename := AssignmentJsonFilename(stackId, userid, setnum)
	tracing = ReadSynapsesJson(jsonFilename)
	log.Println("Read assignment Json:", len(tracing.Data), "synapses")

	// Read in the exported body annotations to determine whether PSD was
	// traced to anchor body or it was orphan/leaves.
	annotations := ReadStackBodyAnnotations(exportedStack)
	log.Println("Read exported bodies Json:", len(annotations), "bodies")

	// For each PSD, find body associated with it using superpixel tiles
	// and the exported session's map.
	var noBodyAnnotated int
	var totalPsds int
	var psdsChanged int // For quality-control: make sure PSDs actually traced

	synapses := (*tracing).Data
	for s, _ := range synapses {
		synapses[s].Tbar.Assignment = fmt.Sprintf("%s-%d",
			StackDescription[stackId], setnum)
		excludeBodies := make(BodySet)
		curPsdBodies := make(BodySet)
		tbarBody, radius := GetNearestBodyOfLocation(&exportedStack,
			synapses[s].Tbar.Location, excludeBodies, curPsdBodies)
		if radius > 0 {
			log.Println("Warning: T-bar", synapses[s].Tbar.Location,
				"was on ZERO SUPERPIXEL but assigned to body",
				tbarBody, "at radius", radius, "from T-bar point")
			synapses[s].Tbar.UsedBodyRadius = radius
		}
		// Make first pass through all PSDs
		excludeBodies[tbarBody] = true
		ambiguous := []int{}
		for p, psd := range synapses[s].Psds {
			totalPsds++
			bodyId := GetBodyOfLocation(&exportedStack, psd.Location)
			if bodyId != GetBodyOfLocation(&baseStack, psd.Location) {
				psdsChanged++
			}
			if bodyId == 0 {
				ambiguous = append(ambiguous, p)
			} else {
				curPsdBodies[bodyId] = true
			}
			bodyNote, found := annotations[bodyId]
			if found {
				_ = addTracedBody(&(synapses[s].Psds[p]), bodyId, bodyNote)
			} else {
				noBodyAnnotated++
				log.Println("Warning: PSD ", psd.Location, " -> ",
					"exported body ", bodyId, " cannot be found in ",
					"body annotation file for exported stack... skipping")
			}
		}
		// Handle ambiguous PSDs, i.e. ones on zero superpixels.
		if len(ambiguous) > 0 {
			for _, p := range ambiguous {
				pPsd := &(synapses[s].Psds[p])
				bodyId, radius := GetNearestBodyOfLocation(&exportedStack,
					(*pPsd).Location, excludeBodies, curPsdBodies)
				if bodyId == 0 {
					(*pPsd).BodyIssue = true
				} else {
					if curPsdBodies[bodyId] {
						log.Println("Flagged: Found body", bodyId, "for PSD",
							(*pPsd).Location, "but it is also assigned to",
							"another PSD.")
					} else {
						log.Println("Found body", bodyId, "for PSD",
							(*pPsd).Location, "after search to radius of",
							radius, "pixels.")
					}
					bodyNote, found := annotations[bodyId]
					if found {
						pTracing := addTracedBody(pPsd, bodyId, bodyNote)
						(*pTracing).UsedBodyRadius = radius
					} else {
						noBodyAnnotated++
						log.Println("Warning: Ambiguous PSD", (*pPsd).Location,
							"-> exported body", bodyId, "cannot be found in",
							"body annotation file for exported stack... skipping")
					}
				}
			}
		}
	}

	if noBodyAnnotated > 0 {
		log.Println("*** PSD bodies not annotated: ", noBodyAnnotated)
	}
	if psdsChanged == 0 {
		log.Println("ERROR: None of", totalPsds,
			"PSD bodies were changed during proofreading!")
		log.Println("  Userid:", userid)
		log.Println("  Stack:", StackDescription[stackId])
		log.Println("  Assignment Set:", setnum)
		log.Println("  Assignment Json:", jsonFilename)
		log.Println("  Exported Stack:", exportedStack)
	} else {
		log.Println("Proofreader altered", psdsChanged, "of", totalPsds,
			"during synapse-driven proofreading")
	}
	return
}

// TransformBodies applies a body->body map to transform any traced bodies.
func (tracing *JsonSynapses) TransformBodies(matchedBodyMap BestOverlapMap,
	stackId StackId) (psdBodies BodySet) {

	psdBodies = make(BodySet)
	numErrors := 0
	altered := 0
	unaltered := 0
	for s, synapse := range (*tracing).Data {
		for p, psd := range synapse.Psds {
			pPsd := &((*tracing).Data[s].Psds[p])
			for t, tracing := range (*pPsd).Tracings {
				if tracing.Result != Orphan && tracing.Result != Leaves &&
					tracing.Result != 0 {

					var origBody BodyId
					if stackId != Target12k {
						origBody = tracing.ExportedBody
					} else {
						origBody = tracing.BaseColumnBody
					}
					match, found := matchedBodyMap[origBody]
					if !found {
						log.Println("ERROR: Body->body map does not contain",
							"body", tracing.Result, "for", tracing.Userid,
							"tracing PSD", psd.Location)
						(*pPsd).TransformIssue = true
						numErrors++
					} else {
						if origBody != match.MatchedBody {
							altered++
							fmt.Println("PSD body", origBody, "->",
								match.MatchedBody)
						} else {
							unaltered++
						}
						switch stackId {
						case Distal, Proximal:
							(*pPsd).Tracings[t].BaseColumnBody = match.MatchedBody
							(*pPsd).Tracings[t].ColumnOverlaps = match.OverlapSize
							(*pPsd).Tracings[t].ExportedSize = match.MaxOverlap
						case Target12k:
							(*pPsd).Tracings[t].Result = TracingResult(match.MatchedBody)
							(*pPsd).Tracings[t].TargetOverlaps = match.OverlapSize
						}
						psdBodies[match.MatchedBody] = true
					}
				}
			}
		}
	}

	if numErrors > 0 {
		log.Println("FATAL ERROR: had", numErrors,
			"errors when transforming PSD bodies.")
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

type psdIndex struct {
	synapseI int
	psdI     int
}

// AddPsdUids modifies a synapse annotation list to include "uid" tags
// for each PSD, either generated from the PSD location or from a matching
// PSD's uid in a given synapse file.
func (tracing *JsonSynapses) AddPsdUids(xformed *JsonSynapses) {
	// If we have a transformed synapse list, create an index using
	// PSD location
	uidMap := make(map[Point3d]psdIndex)
	if xformed != nil {
		for s, synapse := range (*xformed).Data {
			for p, psd := range synapse.Psds {
				uidMap[psd.Location] = psdIndex{s, p}
			}
		}
	}

	// Go through all our PSDs and add uids
	for s, synapse := range (*tracing).Data {
		pSynapse := &((*tracing).Data[s])
		for p, psd := range (*pSynapse).Psds {
			if xformed == nil {
				(*pSynapse).Psds[p].Uid = PsdUid(
					TbarUid(synapse.Tbar.Location), psd.Location)
			} else {
				i, found := uidMap[psd.Location]
				if found {
					(*pSynapse).Psds[p].Uid =
						(*xformed).Data[i.synapseI].Psds[i.psdI].Uid
				} else {
					log.Println("ERROR: No matching transformed PSD found",
						"for PSD", psd.Location)
				}
			}
		}
	}
}

// TransformSynapses modifies synapse locations (T-bar and PSDs) based
// on a transformed synapses annotation list with 'uid' tags for both
// T-bars and PSDs.
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
	for s, synapse := range (*tracing).Data {
		pSynapse := &((*tracing).Data[s])
		// Alter T-bar location
		var uid string
		if synapse.Tbar.Uid == "" {
			uid = TbarUid(synapse.Tbar.Location)
			(*pSynapse).Tbar.Uid = uid
		} else {
			uid = synapse.Tbar.Uid
		}
		i, found := uidMap[uid]
		if !found {
			numTbarErrors++
			log.Printf("** Warning: No tbar uid %s with xformed synapse list!\n",
				uid)
		} else {
			(*pSynapse).Tbar.Location = xformed.Data[i].Tbar.Location
			alteredTbar++

			// Get map of PSDs in transformed T-bar
			xformedPsds := xformed.Data[i].Psds
			xpsdMap := make(map[string]int)
			for xp, xpsd := range xformedPsds {
				if xpsd.Uid == "" {
					log.Printf("** Warning: Xformed PSD %s has no uid!\n",
						xpsd.Location)
				} else {
					xpsdMap[xpsd.Uid] = xp
				}
			}

			// Transform current PSDs by matching xformed PSD uid
			for p, psd := range (*pSynapse).Psds {
				pPsd := &((*pSynapse).Psds[p])
				xp, found := xpsdMap[psd.Uid]
				if found {
					(*pPsd).Location = xformedPsds[xp].Location
					alteredPsds++
				} else {
					log.Printf("** Warning: No match for psd %s, uid %s\n",
						psd.Location, psd.Uid)
					log.Println(" Does not match any of following xformed psds:")
					for _, xpsd := range xformedPsds {
						log.Println("  ", xpsd.Uid, xpsd.Location)
					}
					numPsdErrors++
					(*pPsd).TransformIssue = true
				}
			}
		}
	}

	log.Printf("Transformed locations of %d T-bars and %d PSDs\n",
		alteredTbar, alteredPsds)
	if numTbarErrors > 0 || numPsdErrors > 0 {
		log.Fatalln("FATAL ERROR:", numTbarErrors, "uids unmatched",
			"and", numPsdErrors, "PSDs unmatched using signatures")
	}
	return
}
