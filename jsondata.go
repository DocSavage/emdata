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
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"time"
)

func CreateMetadata(description string) (metadata map[string]interface{}) {
	user, _ := user.Current()
	metadata = make(map[string]interface{})
	metadata["username"] = user.Username
	metadata["date"] = time.Now().Format("02-January-2006 15:04")
	metadata["computer"], _ = os.Hostname()
	metadata["software"] = os.Args[0]
	metadata["description"] = description
	metadata["file version"] = 1 // Necessary for Raveler
	return
}

const (
	JsonSynapseFilename  = "annotations-synapse.json"
	JsonBodyFilename     = "annotations-body.json"
	JsonBookmarkFilename = "annotations-bookmarks.json"
)

// JsonBodies is the high-level structure for an entire
// body annotation list
type JsonBodies struct {
	Metadata map[string]interface{} `json:"metadata"`
	Data     []JsonBody             `json:"data"`
}

// JsonBody is the basic body unit of a body annotation list,
// containing body status, name, etc.
type JsonBody struct {
	Body     BodyId `json:"body ID"`
	Status   string `json:"status"`
	Anchor   string `json:"anchor,omitempty"`
	Name     string `json:"name,omitempty"`
	CellType string `json:"cell type,omitempty"`
	Location string `json:"location,omitempty"`
	Comment  string `json:"comment,omitempty"`
}

// AnchorComment returns true if "Anchor Body" appears in the
// body comments.
func (bodyNote JsonBody) AnchorComment() bool {
	matched, err := regexp.MatchString(".*[Aa]nchor [Bb]ody.*", bodyNote.Comment)
	if err != nil {
		log.Fatalf("FATAL ERROR: AnchorComment(): %s\n", err)
	}
	return matched
}

// OrphanComment returns true if "orphan" appears in the body comments.
func (bodyNote JsonBody) OrphanComment() bool {
	matched, err := regexp.MatchString(".*[Oo]rphan.*", bodyNote.Comment)
	if err != nil {
		log.Fatalf("FATAL ERROR: OrphanComment(): %s\n", err)
	}
	return matched
}

// ReadBodiesJson returns a bodies structure corresponding to 
// a JSON body annotation file.
func ReadBodiesJson(filename string) (bodies *JsonBodies) {
	var file *os.File
	var err error
	if file, err = os.Open(filename); err != nil {
		log.Fatalf("FATAL ERROR: Failed to open JSON file: %s [%s]",
			filename, err)
	}
	dec := json.NewDecoder(file)
	if err := dec.Decode(&bodies); err == io.EOF {
		log.Fatalf("FATAL ERROR: No data in JSON file: %s\n", filename)
	} else if err != nil {
		log.Fatalf("FATAL ERROR: Error reading JSON file (%s): %s\n",
			filename, err)
	}
	return bodies
}

// JsonSynapses is the high-level structure for an entire
// synapse annotation list
type JsonSynapses struct {
	Metadata map[string]interface{}
	Data     []JsonSynapse `json:"data,omitempty"`
}

// WriteJson writes indented JSON synapse annotation list to writer
func (synapses JsonSynapses) WriteJson(writer io.Writer) {
	m, err := json.Marshal(synapses)
	if err != nil {
		log.Fatalf("Error in writing json: %s", err)
	}
	var buf bytes.Buffer
	json.Indent(&buf, m, "", "    ")
	buf.WriteTo(writer)
}

// JsonSynapse holds a T-bar and associated PSDs (partners)
type JsonSynapse struct {
	Tbar JsonTbar  `json:"T-bar"`
	Psds []JsonPsd `json:"partners"`
}

// JsonTbar holds various T-bar attributes including a uid and
// assignment useful for analysis and tracking synapses through
// transformations.
type JsonTbar struct {
	Location   Point3d `json:"location"`
	Body       BodyId  `json:"body ID"`
	Status     string  `json:"status,omitempty"`
	Confidence float32 `json:"confidence,omitempty"`
	Uid        string  `json:"uid,omitempty"`
	Assignment string  `json:"assignment,omitempty"`
}

// JsonPsd holds information for a post-synaptic density (PSD),
// including the tracing results for various proofreading agents.
type JsonPsd struct {
	Location       Point3d       `json:"location"`
	Body           BodyId        `json:"body ID"`
	Confidence     float32       `json:"confidence,omitempty"`
	Uid            string        `json:"uid,omitempty"`
	Tracings       []JsonTracing `json:"tracings"`
	TransformIssue bool          `json:"transform issue,omitempty"`
	BodyIssue      bool          `json:"body issue,omitempty"`
}

// JsonTracing is the data from a single PSD tracing and also
// holds data useful for quality control to determine if
// transformations and overlap analysis was correct.
type JsonTracing struct {
	Userid         string        `json:"userid"`
	Result         TracingResult `json:"result"`
	Stack          string        `json:"stack id"`
	AssignmentSet  int           `json:"assignment set"`
	ExportedBody   BodyId        `json:"exported traced body"`
	ExportedSize   int           `json:"exported traced body size,omitempty"`
	BaseColumnBody BodyId        `json:"base column traced body,omitempty"`
	Orig12kBody    BodyId        `json:"12k traced body,omitempty"`
	ColumnOverlaps int           `json:"export->base overlap,omitempty"`
	TargetOverlaps int           `json:"orig12k->target overlap,omitempty"`
}

// GetTracingIndex returns the index of the PSD given a PSD uid. 
func (synapse JsonSynapse) GetPsdIndex(psdUid string) (index int, found bool) {
	for i, psd := range synapse.Psds {
		if psd.Uid == psdUid {
			return i, true
		}
	}
	return -1, false
}

// TbarUid returns a string T-bar uid for a given 3d point
func TbarUid(pt Point3d) string {
	x, y, z := pt.XYZ()
	return fmt.Sprintf("%05d-%05d-%05d", x, y, z)
}

// PsdUid returns a string PSD uid for a given PSD
func PsdUid(tbarUid string, psdPt Point3d) string {
	x, y, _ := psdPt.XYZ()
	return fmt.Sprintf("%s-psyn-%05d-%05d", tbarUid, x, y)
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
		log.Fatalf("FATAL ERROR: Failed to open JSON file: %s [%s]",
			filename, err)
	}
	dec := json.NewDecoder(file)
	var synapses *JsonSynapses
	if err := dec.Decode(&synapses); err == io.EOF {
		log.Fatalf("FATAL ERROR: No data in JSON file: %s\n", filename)
	} else if err != nil {
		log.Fatalf("FATAL ERROR: Error reading JSON file (%s): %s\n",
			filename, err)
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

// BodyAnnotations correspond to data in a body annotation file
type BodyAnnotations map[BodyId]JsonBody

// ReadStackBodyAnnotations returns the BodyAnnotations for a given stack
func ReadStackBodyAnnotations(stack JsonStack) (annotations BodyAnnotations) {
	annotations = make(BodyAnnotations)
	bodyNotes := ReadBodiesJson(stack.StackBodiesJsonFilename())
	for _, bodyNote := range bodyNotes.Data {
		annotations[bodyNote.Body] = bodyNote
	}
	return
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
