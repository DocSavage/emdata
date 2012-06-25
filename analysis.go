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
    "os"
    "log"
    "encoding/json"
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

// TracingAgent is a unique id that describes a proofreading agent.
type TracingAgent string

// Tracings holds the results of agents.
type Tracings map[TracingAgent]TracingResult

// PsdTracing holds the results of a single agent tracing PSDs.
type PsdTracing map[Point3d]TracingResult

func (tracing PsdTracing) WriteJson(filename string, 
    bodyToBodyMap map[BodyId]BodyId) {

    file, err := os.Open(filename)
    if err != nil {
        log.Fatalf("Failed to open file for writing JSON of PSD tracing: %s\n", 
            filename)
    }
    enc := json.NewEncoder(file)
    if err = enc.Encode(&tracing); err != nil {
        log.Fatalf("Could not encoding PSD tracing: %s", err)
    }
}

// PsdTracings holds the results of agents tracing PSDs.
type PsdTracings map[Point3d]Tracings


