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
	"encoding/json"
	"io"
	"log"
	"os"
	"path/filepath"
)

const (
	JsonSynapseFilename  = "annotations-synapse.json"
	JsonBodyFilename     = "annotations-body.json"
	JsonBookmarkFilename = "annotations-bookmarks.json"
)

// JsonBodies is the high-level structure for an entire
// body annotation list
type JsonBodies struct {
	Metadata map[string]interface{}
	Data     []JsonBody `json:"data"`
}

type JsonBody struct {
	Body     BodyId `json:"body ID"`
	Status   string `json:"status"`
	Anchor   string `json:"anchor,omitempty"`
	Name     string `json:"name,omitempty"`
	CellType string `json:"cell type,omitempty"`
	Location string `json:"location,omitempty"`
	Comment  string `json:"comment,omitempty"`
}

// ReadBodiesJson returns a bodies structure corresponding to 
// a JSON body annotation file.
func ReadBodiesJson(filename string) (bodies *JsonBodies) {
	var file *os.File
	var err error
	if file, err = os.Open(filename); err != nil {
		log.Fatalf("Failed to open JSON file: %s [%s]",
			filename, err)
	}
	dec := json.NewDecoder(file)
	if err := dec.Decode(bodies); err == io.EOF {
		log.Fatalf("No data in JSON file: %s\n", filename)
	} else if err != nil {
		log.Fatalf("Error reading JSON file (%s): %s\n", filename, err)
	}
	return bodies
}

// JsonSynapses is the high-level structure for an entire
// synapse annotation list
type JsonSynapses struct {
	Metadata map[string]interface{}
	Data     []JsonSynapse `json:"data,omitempty"`
}

type JsonSynapse struct {
	Tbar JsonTbar  `json:"T-bar"`
	Psds []JsonPsd `json:"partners"`
}

type JsonTbar struct {
	Location   Point3d `json:"location"`
	Body       BodyId  `json:"body ID"`
	Status     string  `json:"status,omitempty"`
	Confidence float32 `json:"confidence,omitempty"`
}

type JsonPsd struct {
	Location   Point3d `json:"location"`
	Body       BodyId  `json:"body ID"`
	Confidence float32 `json:"confidence,omitempty"`
}

// StackSynapsesJsonFilename returns the file name of the
// synapse annotation file for a given stack directory
func StackSynapsesJsonFilename(stackPath string) string {
	return filepath.Join(stackPath, JsonSynapseFilename)
}

// StackBodiesJsonFilename returns the file name of the
// body annotation file for a given stack directory
func StackBodiesJsonFilename(stackPath string) string {
	return filepath.Join(stackPath, JsonBodyFilename)
}

// ReadSynapsesJson returns a synapse structure corresponding to 
// a JSON synapse annotation file.
func ReadSynapsesJson(filename string) *JsonSynapses {
	var file *os.File
	var err error
	if file, err = os.Open(filename); err != nil {
		log.Fatalf("Failed to open JSON file: %s [%s]",
			filename, err)
	}
	dec := json.NewDecoder(file)
	var synapses *JsonSynapses
	if err := dec.Decode(&synapses); err == io.EOF {
		log.Fatalf("No data in JSON file: %s\n", filename)
	} else if err != nil {
		log.Fatalf("Error reading JSON file (%s): %s\n", filename, err)
	}
	return synapses
}

// JsonStack is a stack that contains synapse, 
// body, and other JSON files that pure sessions directories would 
// keep in a session pickle file.
type JsonStack interface {
	StackSynapsesJsonFilename() string
	StackBodiesJsonFilename() string
}

// ReadStackBodiesJson returns the default body annotation file
// for a given stack.
func ReadStackBodiesJson(stack JsonStack) *JsonBodies {
	return ReadBodiesJson(stack.StackBodiesJsonFilename())
}

// ReadStackBodyAnnotations returns the default body annotation file
// for a given stack.
func ReadStackBodyAnnotations(stack JsonStack) (
	bodyToNotesMap map[BodyId]JsonBody) {

	bodyToNotesMap = make(map[BodyId]JsonBody)
	bodyNotes := ReadBodiesJson(stack.StackBodiesJsonFilename())
	for _, bodyNote := range bodyNotes {
		bodyToNotesMap[bodyNote.Body] = bodyNote
	}
	return bodyToNotesMap
}

// ReadStackSynapsesJson returns the default synapse annotation file
// for a given stack.
func ReadStackSynapsesJson(stack JsonStack) *JsonSynapses {
	return ReadSynapsesJson(stack.StackSynapsesJsonFilename())
}

// ReadPsdBodyMap returns a PSD -> Body Id map from a
// stack's synapse annotation file.
func ReadPsdBodyMap(stack JsonStack) LocationToBodyMap {
	synapses := ReadStackSynapsesJson(stack)
	psdToBodyMap := make(LocationToBodyMap)
	for _, synapse := range synapses.Data {
		for _, psd := range synapse.Psds {
			psdToBodyMap[psd.Location] = psd.Body
		}
	}
	return psdToBodyMap
}
