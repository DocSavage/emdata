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
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
)

// Connectome holds the strength of connections between two body ids
// in a directed fashion.  The first key is the pre-synaptic body
// and the second is the post-synaptic body id.
type Connectome map[BodyId](map[BodyId]int)

// AddConnection adds a (pre, post) connection of given strength
// to a connectome.
func (c *Connectome) AddConnection(pre, post BodyId, strength int) {
	if len(*c) == 0 {
		*c = make(Connectome)
	}
	connections, preFound := (*c)[pre]
	if preFound {
		_, postFound := connections[post]
		if postFound {
			(*c)[pre][post] += strength
		} else {
			(*c)[pre][post] = strength
		}
	} else {
		(*c)[pre] = make(map[BodyId]int)
		(*c)[pre][post] = strength
	}
}

// Add returns a connectome that's the sum of two connectomes.
func (c1 Connectome) Add(c2 Connectome) (sum Connectome) {
	sum = make(Connectome)
	for body1, connections := range c1 {
		sum[body1] = make(map[BodyId]int)
		for body2, strength := range connections {
			sum[body1][body2] = strength
		}
	}
	for body1, connections := range c2 {
		for body2, strength := range connections {
			sum.AddConnection(body1, body2, strength)
		}
	}
	return
}

// WriteMatlab writes connectome data as Matlab code for a
// containers.Map() data structure.  Key names are body names
// within the passed NamedBodyMap.
func (c Connectome) WriteMatlab(writer io.Writer, connectomeName string,
	namedBodyMap NamedBodyMap) {

	bufferedWriter := bufio.NewWriter(writer)
	defer bufferedWriter.Flush()

	_, err := fmt.Fprintf(bufferedWriter, "%s = containers.Map()\n",
		connectomeName)
	if err != nil {
		log.Fatalf("ERROR: Unable to write matlab code: %s", err)
	}
	for bodyId1, namedBody1 := range namedBodyMap {
		for bodyId2, namedBody2 := range namedBodyMap {
			key := namedBody1.Name + "," + namedBody2.Name
			connections, preFound := c[bodyId1]
			if preFound {
				strength, postFound := connections[bodyId2]
				if postFound {
					_, err := fmt.Fprintf(bufferedWriter, "%s('%s') = %d\n",
						connectomeName, key, strength)
					if err != nil {
						log.Fatalln("ERROR: Unable to write matlab code:",
							err)
					}
				}
			}
		}
	}
}

// WriteMatlabFile writes connectome data as Matlab code for a
// containers.Map() data structure into the given filename.
func (c Connectome) WriteMatlabFile(filename string, connectomeName string,
	namedBodyMap NamedBodyMap) {

	file, err := os.Create(filename)
	if err != nil {
		log.Fatalf("FATAL ERROR: Failed to create connectome matlab file: %s [%s]\n",
			filename, err)
	}
	c.WriteMatlab(file, connectomeName, namedBodyMap)
	file.Close()
}

// WriteCsv writes connectome data in CSV format with body names as
// headers for rows/columns
func (c Connectome) WriteCsv(writer io.Writer, namedBodyMap NamedBodyMap) {

	csvWriter := csv.NewWriter(writer)

	// Print body names along first row
	numBodies := len(namedBodyMap)
	numCells := numBodies + 1 // Leave 1 cell for header of row/col
	record := make([]string, numCells)
	n := 1
	for _, namedBody := range namedBodyMap {
		record[n] = namedBody.Name
		n++
	}
	err := csvWriter.Write(record)
	if err != nil {
		log.Fatalln("ERROR: Unable to write body names as CSV:", err)
	}

	// For every subsequent row, the first column is body name,
	// and the rest are the strengths of (pre, post) where pre body
	// name is listed in 1st column.
	for bodyId1, namedBody1 := range namedBodyMap {
		record[0] = namedBody1.Name
		n := 1
		for bodyId2, _ := range namedBodyMap {
			strength := 0
			connections, preFound := c[bodyId1]
			if preFound {
				value, postFound := connections[bodyId2]
				if postFound {
					strength = value
				}
			}
			record[n] = strconv.Itoa(strength)
			n++
		}
		err := csvWriter.Write(record)
		if err != nil {
			log.Fatalln("ERROR: Unable to write line of CSV for ",
				"presynaptic body", namedBody1.Name, ":", err)
		}
	}
}

// WriteCsvFile writes connectome data into a CSV file.
func (c Connectome) WriteCsvFile(filename string, namedBodyMap NamedBodyMap) {
	file, err := os.Create(filename)
	if err != nil {
		log.Fatalf("ERROR: Failed to create connectome csv file: %s [%s]\n",
			filename, err)
	}
	c.WriteCsv(file, namedBodyMap)
	file.Close()
}
