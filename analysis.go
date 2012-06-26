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
    "encoding/json"
    "io"
    "log"
//    "os"
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

func (tracing PsdTracing) WriteJson(writer io.Writer, agent TracingAgent) {

    enc := json.NewEncoder(writer)

    var trace JsonAgentTracing
    trace.Agent = string(agent)

    var jsonTracings JsonPsdTracings
    jsonTracings.Metadata = CreateMetadata("PSD Tracing")
    jsonTracings.Data = make([]JsonPsdTracing, len(tracing))
    i := 0
    for location, result := range tracing {
        jsonTracings.Data[i].Location = location
        trace.Result = result.String()
        jsonTracings.Data[i].Tracings = []JsonAgentTracing{ trace }
    }
    if err := enc.Encode(&jsonTracings); err != nil {
        log.Fatalf("Could not encode PSD tracing: %s", err)
    }
}

// CreatePsdTracing creates a PsdTracing struct by transforming assignment tracings
// from one stack to another, assuming that watersheds are mostly preserved and
// using maximal watershed overlap to determine equivalence among bodies.
func CreatePsdTracing(assignmentJsonFilename string, exportSessionDir string, 
    baseStackDir string, targetStackDir string) (tracing PsdTracing) {

    // Set these directories to appropriate Raveler stack types.
    var baseStack BaseStack
    baseStack.Directory = baseStackDir

    var exportedStack ExportedStack
    exportedStack.Directory = exportSessionDir
    exportedStack.Base = baseStack

    var targetStack BaseStack
    targetStack.Directory = targetStackDir

    // Read in the assignment JSON: set of PSDs
    assignment := ReadSynapsesJson(assignmentJsonFilename)
    fmt.Println("Read assignment Json:", len(assignment.Data), "synapses")

    // Read in the exported body annotations to determine whether PSD was
    // traced to anchor body or it was orphan/leaves.
    bodyToNotesMap := ReadStackBodyAnnotations(exportedStack)
    fmt.Println("Read exported bodies Json:", len(bodyToNotesMap), "bodies")

    // For each PSD, find body associated with it using superpixel tiles
    // and the exported session's map.
    definitiveAnchor := 0
    commentedAnchor := 0
    commentedOrphan := 0
    presumedLeaves := 0
    noBodyAnnotated := 0
    tracing = make(PsdTracing)
    psdBodies := make(map[BodyId]bool)  // Set of PSD bodies

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
                fmt.Println("WARNING!!! PSD ", psd.Location, " -> ",
                    "exported body ", bodyId, " cannot be found in ",
                    "body annotation file for exported stack... skipping")
            }
        }
    }

    fmt.Println("  Anchors marked with anchor tag: ", definitiveAnchor)
    fmt.Println(" Anchors detected within comment: ", commentedAnchor)
    fmt.Println(" Orphans detected within comment: ", commentedOrphan)
    fmt.Println("PSD bodies with no anchor/orphan: ", presumedLeaves)
    if noBodyAnnotated > 0 {
        fmt.Println("\n*** PSD bodies not annotated: ", noBodyAnnotated)
    }

    // For each PSD body, determine the exported session superpixels
    // within that body.
    bodyToSpMap := exportedStack.BodySuperpixels(psdBodies)

    // Determine which bodies in target stack have maximal overlap
    // with the PSD bodies based on superpixels
    bodyToBodyMap := targetStack.OverlapAnalysis(bodyToSpMap)

    // Finalize the PSD Tracing by transforming traced session body ids into
    // target stack body ids using the body->body map.
    numErrors := 0
    altered := 0
    for location, result := range tracing {
        if result != Orphan && result != Leaves && result != 0 {
            targetBody, found := bodyToBodyMap[BodyId(result)]
            if !found {
                log.Println("ERROR!!! Unable to find target body corresponding to ",
                    "session body ", result)
                numErrors++
            } else {
                tracing[location] = TracingResult(targetBody)
                altered++
            }
        }
    }
    if numErrors > 0 {
        log.Fatalln("\nAborting... found ", numErrors, " when converting to target stack")
    }
    fmt.Printf("\nTransformed %d of %d body targets in traced PSDs\n",
        altered, len(tracing))
    return
}


// PsdTracings holds the results of agents tracing PSDs.
type PsdTracings map[Point3d]Tracings



