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
	"sort"
	"strconv"
	"strings"
)

type Connection struct {
	Pre      string
	Post     string
	Strength int
}

type ConnectionList []Connection

func (list ConnectionList) Len() int           { return len(list) }
func (list ConnectionList) Swap(i, j int)      { list[i], list[j] = list[j], list[i] }
func (list ConnectionList) Less(i, j int) bool { return list[i].Strength > list[j].Strength }

// SortByStrength sorts a ConnectionList in descending order of strength
func (list ConnectionList) SortByStrength() {
	sort.Sort(list)
}

// Connectome holds the strength of connections between two body ids
// in a directed fashion.  The first key is the pre-synaptic body
// and the second is the post-synaptic body id.
type Connectome map[BodyId](map[BodyId]int)

// GetConnection returns a (pre, post) strength and 'found' bool.
func (c Connectome) ConnectionStrength(pre, post BodyId) (strength int, found bool) {
	connections, found := c[pre]
	if found {
		_, found = connections[post]
		if found {
			strength = c[pre][post]
			if strength == 0 {
				found = false
			}
		}
	}
	return
}

// NamedConnectome holds strength of connections between two bodies
// that are identified using names (strings) instead of body ids as
// in the Connectome type.
type NamedConnectome map[string](map[string]int)

// GetConnection returns a (pre, post) strength and 'found' bool.
func (nc NamedConnectome) ConnectionStrength(pre, post string) (strength int, found bool) {
	connections, found := nc[pre]
	if found {
		_, found = connections[post]
		if found {
			strength = nc[pre][post]
			if strength == 0 {
				found = false
			}
		}
	}
	return
}

func (nc NamedConnectome) MatchingNames(patterns []string) (matches []string) {
	matches = make([]string, 0, len(patterns))
	for _, pattern := range patterns {
		if pattern[len(pattern)-1:] == "*" {
			// Use as prefix
			pattern = pattern[:len(pattern)-1]
			for name, _ := range nc {
				if strings.HasPrefix(name, pattern) {
					matches = append(matches, name)
				}
			}
		} else {
			// Require exact matching
			_, found := nc[pattern]
			if found {
				matches = append(matches, pattern)
			}
		}
	}
	return
}

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
	namedBodyList := namedBodyMap.SortByName()
	for _, namedBody1 := range namedBodyList {
		for _, namedBody2 := range namedBodyList {
			key := namedBody1.Name + "," + namedBody2.Name
			connections, preFound := c[namedBody1.Body]
			if preFound {
				strength, postFound := connections[namedBody2.Body]
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

// Python code for Neuoptikon
const neuroptikonHeader = `
def findOrCreateLocation(location):
	region = network.findRegion(name = location)
	if not region:
		region = network.createRegion(name = location)
	return region

def findOrCreateBody(bodyName, regionName=None):
    neuron = network.findNeuron(name = bodyName)
    if not neuron:
        neuron = network.createNeuron(name = bodyName)
    if regionName:
    	region = findOrCreateLocation(regionName)


    return neuron

neurons = {}
`

const connectionCode = `
connection = pre.SynapseOn(post)
connection.addAttribute('Count', Attribute.INTEGER_TYPE, int(%d))
`

// WriteNeuroptikon writes connectome data in a python script that can be
// executed by the Neuroptikon program
func (c Connectome) WriteNeuroptikon(writer io.Writer, namedBodyMap NamedBodyMap) {

	bufferedWriter := bufio.NewWriter(writer)
	defer bufferedWriter.Flush()

	_, err := fmt.Fprintln(bufferedWriter, neuroptikonHeader)
	if err != nil {
		log.Fatalf("ERROR: Unable to write Neuroptikon code: %s", err)
	}

	namedBodyList := namedBodyMap.SortByName()

	for _, namedBody1 := range namedBodyList {
		for _, namedBody2 := range namedBodyList {
			connections, preFound := c[namedBody1.Body]
			if preFound {
				strength, postFound := connections[namedBody2.Body]
				if postFound {
					_, err := fmt.Fprintf(bufferedWriter,
						"pre = findOrCreateBody('%s', '%s')\n",
						namedBody1.Name, namedBody1.Location)
					if err != nil {
						log.Fatalln("ERROR: Unable to write python code:", err)
					}
					_, err = fmt.Fprintf(bufferedWriter,
						"post = findOrCreateBody('%s', '%s')\n",
						namedBody2.Name, namedBody2.Location)
					if err != nil {
						log.Fatalln("ERROR: Unable to write python code:", err)
					}
					_, err = fmt.Fprintf(bufferedWriter, connectionCode, strength)
					if err != nil {
						log.Fatalln("ERROR: Unable to write python code:", err)
					}
				}
			}
		}
	}
}

// WriteNeuroptikonFile writes connectome data into a python for Neuroptikon import
func (c Connectome) WriteNeuroptikonFile(filename string, namedBodyMap NamedBodyMap) {
	file, err := os.Create(filename)
	if err != nil {
		log.Fatalf("ERROR: Failed to create connectome Neuroptikon file: %s [%s]\n",
			filename, err)
	}
	c.WriteNeuroptikon(file, namedBodyMap)
	file.Close()
}

// WriteCsv writes connectome data in CSV format with body names as
// headers for rows/columns
func (c Connectome) WriteCsv(writer io.Writer, namedBodyMap NamedBodyMap) {

	csvWriter := csv.NewWriter(writer)
	namedBodyList := namedBodyMap.SortByName()

	// Print body names along first row
	numBodies := len(namedBodyList)
	numCells := numBodies + 1 // Leave 1 cell for header of row/col
	record := make([]string, numCells)
	n := 1
	for _, namedBody := range namedBodyList {
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
	for _, namedBody1 := range namedBodyList {
		record[0] = namedBody1.Name
		n := 1
		for _, namedBody2 := range namedBodyList {
			strength := 0
			connections, preFound := c[namedBody1.Body]
			if preFound {
				value, postFound := connections[namedBody2.Body]
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

// WriteCsv writes connectome data in CSV format with body names as
// headers for rows/columns
func ReadCsv(reader io.Reader) (nc *NamedConnectome) {
	nc = new(NamedConnectome)
	csvReader := csv.NewReader(reader)

	// Read the body names in first row.
	bodyNames, err := csvReader.Read()
	if err == io.EOF {
		log.Fatalln("ERROR: Unable to read first line of connectome CSV:",
			err)
	}

	// Read all connectivity matrix
	for {
		items, err := csvReader.Read()
		if err == io.EOF {
			break
		} else if err != nil {
			log.Println("Warning:", err)
		} else if items[0] == "" {
			continue
		} else if len(items) != len(bodyNames) {
			log.Fatalf("ERROR: CSV has inconsistent # of columns (%d vs %d)!",
				len(bodyNames), len(items))
		} else {
			preName := items[0]
			for i := 1; i < len(items); i++ {
				postName := bodyNames[i]
				strength, err := strconv.Atoi(items[i])
				if err != nil {
					log.Fatalln("ERROR: Could not parse CSV line:",
						items, "\nError:", err)
				}
				nc.AddConnection(preName, postName, strength)
			}
		}
	}
	return
}

// WriteCsvFile writes connectome data into a CSV file.
func ReadCsvFile(filename string) (nc *NamedConnectome) {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("ERROR: Failed to open connectome csv file: %s [%s]\n",
			filename, err)
	}
	defer file.Close()
	nc = ReadCsv(file)
	return
}

// AddConnection adds a (pre, post) connection of given strength
// to a connectome.
func (nc *NamedConnectome) AddConnection(pre, post string, strength int) {
	if len(*nc) == 0 {
		*nc = make(NamedConnectome)
	}
	connections, preFound := (*nc)[pre]
	if preFound {
		_, postFound := connections[post]
		if postFound {
			(*nc)[pre][post] += strength
		} else {
			(*nc)[pre][post] = strength
		}
	} else {
		(*nc)[pre] = make(map[string]int)
		(*nc)[pre][post] = strength
	}
}
