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
	"bytes"
	"io"
	"encoding/json"
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

// Tracings holds the results of agents.
type Tracings map[TracingAgent]TracingResult

// PsdTracing holds the results of a single agent tracing PSDs.
type PsdTracing map[Point3d]TracingResult

// CreatePsdTracing creates a PsdTracing struct by examining each assigned
// location and determining the exported body ID of the stack for that location.
func CreatePsdTracing(assignmentJsonFilename string, exportedStack ExportedStack) (
	tracing PsdTracing, psdBodies BodySet) {

	// Read in the assignment JSON: set of PSDs
	assignment := ReadSynapsesJson(assignmentJsonFilename)
	log.Println("Read assignment Json:", len(assignment.Data), "synapses")

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
	tracing = make(PsdTracing)
	psdBodies = make(BodySet) // Set of PSD bodies

	for _, synapse := range assignment.Data {
		for _, psd := range synapse.Psds {
			bodyId := GetBodyOfLocation(&exportedStack, psd.Location)
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
				tracing[psd.Location] = tracingResult
			} else {
				noBodyAnnotated++
				log.Println("WARNING!!! PSD ", psd.Location, " -> ",
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
		log.Println("\n*** PSD bodies not annotated: ", noBodyAnnotated)
	}
	return
}

func (tracing PsdTracing) WriteJson(writer io.Writer, agent TracingAgent) {
	/* enc := json.NewEncoder(writer)*/

	var trace JsonAgentTracing
	trace.Agent = string(agent)

	var jsonTracings JsonPsdTracings
	jsonTracings.Metadata = CreateMetadata("PSD Tracing")
	jsonTracings.Data = make([]JsonPsdTracing, len(tracing))
	i := 0
	for location, result := range tracing {
		jsonTracings.Data[i].Location = location
		trace.Result = result.String()
		jsonTracings.Data[i].Tracings = []JsonAgentTracing{trace}
		i++
	}
	m, _ := json.Marshal(jsonTracings)
	var buf bytes.Buffer
	json.Indent(&buf, m, "", "    ")
	buf.WriteTo(writer)
	/*
		if err := enc.Encode(&jsonTracings); err != nil {
			log.Fatalf("Could not encode PSD tracing: %s", err)
		}
	*/
}

// TransformBodies applies a body->body map to transform any traced bodies.
func (tracing *PsdTracing) TransformBodies(bodyToBodyMap map[BodyId]BodyId) {
	numErrors := 0
	altered := 0
	unaltered := 0
	for location, result := range *tracing {
		if result != Orphan && result != Leaves && result != 0 {
			origBody := BodyId(result)
			targetBody, found := bodyToBodyMap[origBody]
			if !found {
				log.Println("ERROR!!! Body->body map does not contain PSD",
					"body ", result)
				numErrors++
			} else if origBody != targetBody {
				(*tracing)[location] = TracingResult(targetBody)
				altered++
			} else {
				unaltered++
			}
		}
	}
	if numErrors > 0 {
		log.Fatalln("\nAborting... found ", numErrors,
			" when transforming PSD bodies.")
	}
	log.Printf("\nTransformed %d of %d PSD bodies\n", altered, altered+unaltered)
}

// PsdTracings holds the results of agents tracing PSDs.
type PsdTracings map[Point3d]Tracings
