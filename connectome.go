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
	"bytes"
	"encoding/csv"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

type Synapse struct {
	Pre  JsonTbar
	Post JsonPsd
}

type Connection []Synapse

func (c Connection) Strength() int {
	return len(c)
}

func (c Connection) WriteNeuroptikon(writer io.Writer) {
	for _, synapse := range c {
		_, err := fmt.Fprintf(writer, "addConnection(pre, post, %d, %s, %s)\n",
			1, synapse.Pre.Location.String(), synapse.Post.Location.String())
		if err != nil {
			log.Fatalln("ERROR: Unable to write python code:", err)
		}
	}
}

type NamedConnection struct {
	Connection
	PreName  string
	PostName string
}

type ConnectionList []NamedConnection

func (list ConnectionList) Len() int {
	return len(list)
}
func (list ConnectionList) Swap(i, j int) {
	list[i], list[j] = list[j], list[i]
}
func (list ConnectionList) Less(i, j int) bool {
	return list[i].Strength() > list[j].Strength()
}

// SortByStrength sorts a ConnectionList in descending order of strength
func (list ConnectionList) SortByStrength() {
	sort.Sort(list)
}

// ConnectivityMap holds the connection data between two body ids
// in a directed fashion.  The first key is the pre-synaptic body
// and the second is the post-synaptic body id.
type ConnectivityMap map[BodyId](map[BodyId]Connection)

// Connectome holds both a catalog of neurons and their connectivity.
type Connectome struct {
	Neurons      NamedBodyMap
	Connectivity ConnectivityMap
}

// WriteGob writes connectome data in Go Gob format
func (c Connectome) WriteGob(writer io.Writer) {
	enc := gob.NewEncoder(writer)
	err := enc.Encode(c)
	if err != nil {
		log.Fatalf("Error in writing connectome gob: %s", err)
	}
}

// WriteGobFile writes connectome data into a Gob file.
func (c Connectome) WriteGobFile(filename string) {
	file, err := os.Create(filename)
	if err != nil {
		log.Fatalf("ERROR: Failed to create connectome Go Gob file: %s [%s]\n",
			filename, err)
	}
	c.WriteGob(file)
	file.Close()
}

// ReadGob reads a connectome from Gob format
func ReadGob(reader io.Reader) (c *Connectome) {
	dec := gob.NewDecoder(reader)
	err := dec.Decode(c)
	if err != nil {
		log.Fatalf("Error in reading connectom gob: %s", err)
	}
	return
}

// ReadGobFile writes connectome data into a CSV file.
func ReadGobFile(filename string) (c *Connectome) {
	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("ERROR: Failed to open connectome Gob file: %s [%s]\n",
			filename, err)
	}
	defer file.Close()
	c = ReadGob(file)
	return
}

type jsonConnectome struct {
	neurons      NamedBodyList
	connectivity jsonConnectivityMap
}
type jsonConnectionMap map[string]Connection
type jsonConnectivityMap map[string]jsonConnectionMap

// WriteJson writes connectome data in JSON format
func (c Connectome) WriteJson(writer io.Writer) {
	// Create a JSON-able structure that has only string keys
	var jsonC jsonConnectome
	jsonC.neurons = c.Neurons.SortByName()
	jsonC.connectivity = make(jsonConnectivityMap)

	for preId, connections := range c.Connectivity {
		pre := fmt.Sprintf("Body %d", preId)
		jsonC.connectivity[pre] = make(map[string]Connection,
			len(connections))
		for postId, connection := range connections {
			post := fmt.Sprintf("Body %d", postId)
			jsonC.connectivity[pre][post] = connection
		}
	}
	log.Println("Json connectivity map has", len(jsonC.connectivity),
		"rows")

	// Write the temporary structure
	m, err := json.Marshal(jsonC)
	if err != nil {
		log.Fatalf("Error in writing connectome json: %s", err)
	}
	var buf bytes.Buffer
	json.Indent(&buf, m, "", "    ")
	buf.WriteTo(writer)
}

// WriteJsonFile writes connectome data into a JSON file.
func (c Connectome) WriteJsonFile(filename string) {
	file, err := os.Create(filename)
	if err != nil {
		log.Fatalf("ERROR: Failed to create connectome JSON file: %s [%s]\n",
			filename, err)
	}
	c.WriteJson(file)
	file.Close()
}

// ConnectionsSortedByName returns a sorted list of NamedConnection
func (c Connectome) ConnectionsSortedByName() (list ConnectionList) {
	list = make(ConnectionList, 0, len(c.Neurons))
	namedBodyList := c.Neurons.SortByName()
	for _, namedBody1 := range namedBodyList {
		for _, namedBody2 := range namedBodyList {
			connections, preFound := c.Connectivity[namedBody1.Body]
			if preFound {
				connection, postFound := connections[namedBody2.Body]
				if postFound {
					list = append(list, NamedConnection{connection,
						namedBody1.Name, namedBody2.Name})
				}
			}
		}
	}
	return
}

// GetConnection returns a (pre, post) strength and 'found' bool.
func (c Connectome) ConnectionStrength(pre, post BodyId) (
	strength int, found bool) {

	strength = 0
	connections, found := c.Connectivity[pre]
	if found {
		connection, found := connections[post]
		if found {
			strength = connection.Strength()
			if strength == 0 {
				found = false
			}
		}
	}
	return
}

// AddSynapse adds a synapse to a given connectome.
func (c *Connectome) AddSynapse(s *Synapse) {
	if len(c.Connectivity) == 0 {
		c.Connectivity = make(ConnectivityMap)
	}
	preId := s.Pre.Body
	postId := s.Post.Body
	connections, preFound := c.Connectivity[preId]
	if preFound {
		_, postFound := connections[postId]
		if postFound {
			c.Connectivity[preId][postId] = append(
				c.Connectivity[preId][postId], *s)
		} else {
			c.Connectivity[preId][postId] = []Synapse{*s}
		}
	} else {
		c.Connectivity[preId] = make(map[BodyId]Connection)
		c.Connectivity[preId][postId] = []Synapse{*s}
	}
}

/*
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
*/

// WriteMatlab writes connectome data as Matlab code for a
// containers.Map() data structure.  Key names are body names
// within the passed NamedBodyMap.
func (c Connectome) WriteMatlab(writer io.Writer, connectomeName string) {

	bufferedWriter := bufio.NewWriter(writer)
	defer bufferedWriter.Flush()

	_, err := fmt.Fprintf(bufferedWriter, "%s = containers.Map()\n",
		connectomeName)
	if err != nil {
		log.Fatalf("ERROR: Unable to write matlab code: %s", err)
	}
	namedBodyList := c.Neurons.SortByName()
	for _, namedBody1 := range namedBodyList {
		preId := namedBody1.Body
		for _, namedBody2 := range namedBodyList {
			postId := namedBody2.Body
			key := namedBody1.Name + "," + namedBody2.Name
			strength, found := c.ConnectionStrength(preId, postId)
			if found {
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

// WriteMatlabFile writes connectome data as Matlab code for a
// containers.Map() data structure into the given filename.
func (c Connectome) WriteMatlabFile(filename string, connectomeName string) {

	file, err := os.Create(filename)
	if err != nil {
		log.Fatalf("FATAL ERROR: Failed to create connectome matlab file: %s [%s]\n",
			filename, err)
	}
	c.WriteMatlab(file, connectomeName)
	file.Close()
}

// Python code for Neuoptikon
const headerCode = `
import library.neuron_class

CellTypes = {}

def findOrCreateLocation(location):
	region = network.findRegion(name=location)
	if not region:
		region = network.createRegion(name=location)
	return region

def findOrCreateBody(bodyName, bodyId, cellType=None, regionName=None,
	primary=False, secondary=False, center=None):

	global CellTypes
	cell = None
	if cellType:
		if cellType in CellTypes:
			cell = CellTypes[cellType]
		else:
			cell = library.neuron_class.NeuronClass(identifier=cellType,
				name=cellType, abbreviation=cellType)
			CellTypes[cellType] = cell

    neuron = network.findNeuron(name=bodyName)
    if not neuron:
	    if regionName:
	    	region = findOrCreateLocation(regionName)
	    	neuron = network.createNeuron(name=bodyName, neuronClass=cell, region=region)
	    else:
	    	neuron = network.createNeuron(name=bodyName, neuronClass=cell, region=None)
	    neuron.addAttribute('BodyID', Attribute.INTEGER_TYPE, bodyId)
	    neuron.addAttribute('Primary', Attribute.BOOLEAN_TYPE, primary)
	    neuron.addAttribute('Secondary', Attribute.BOOLEAN_TYPE, secondary)
	    if center:
		    neuron.addAttribute('CenterX', Attribute.INTEGER_TYPE, center[0])
		    neuron.addAttribute('CenterY', Attribute.INTEGER_TYPE, center[1])
		    neuron.addAttribute('CenterZ', Attribute.INTEGER_TYPE, center[2])
        display.setLabel(neuron, bodyName)

    return neuron

def addConnection(pre, post, strength, tbarCoord, psdCoord):
	connection = pre.synapseOn(post)
	connection.addAttribute('Count', Attribute.INTEGER_TYPE, strength)
	connection.addAttribute('TbarX', Attribute.INTEGER_TYPE, tbarCoord[0])
	connection.addAttribute('TbarY', Attribute.INTEGER_TYPE, tbarCoord[1])
	connection.addAttribute('TbarZ', Attribute.INTEGER_TYPE, tbarCoord[2])
	connection.addAttribute('PsdX', Attribute.INTEGER_TYPE, psdCoord[0])
	connection.addAttribute('PsdY', Attribute.INTEGER_TYPE, psdCoord[1])
	connection.addAttribute('PsdZ', Attribute.INTEGER_TYPE, psdCoord[2])

neurons = {}

network.setBulkLoading(True)
`

const endCode = `
network.setBulkLoading(False)
`

// WriteNeuroptikon writes connectome data in a python script that can be
// executed by the Neuroptikon program
func (c Connectome) WriteNeuroptikon(writer io.Writer) {

	bufferedWriter := bufio.NewWriter(writer)
	defer bufferedWriter.Flush()

	_, err := fmt.Fprintln(bufferedWriter, headerCode)
	if err != nil {
		log.Fatalf("ERROR: Unable to write Neuroptikon code: %s", err)
	}

	for bodyId1, connections := range c.Connectivity {
		namedBody1 := c.Neurons[bodyId1]
		for bodyId2, connection := range connections {
			namedBody2 := c.Neurons[bodyId2]

			fmt.Fprintln(bufferedWriter, "# Body", bodyId1,
				namedBody1.Body, namedBody1.Name, "->",
				bodyId2, namedBody2.Body, namedBody2.Name)
			namedBody1.WriteNeuroptikon(bufferedWriter, true)
			namedBody2.WriteNeuroptikon(bufferedWriter, false)
			connection.WriteNeuroptikon(bufferedWriter)
		}
	}

	_, err = fmt.Fprintln(bufferedWriter, endCode)
	if err != nil {
		log.Fatalf("ERROR: Unable to write Neuroptikon code: %s", err)
	}
}

// WriteNeuroptikonFile writes connectome data into a python for Neuroptikon import
func (c Connectome) WriteNeuroptikonFile(filename string) {
	file, err := os.Create(filename)
	if err != nil {
		log.Fatalf("ERROR: Failed to create connectome Neuroptikon file: %s [%s]\n",
			filename, err)
	}
	c.WriteNeuroptikon(file)
	file.Close()
}

// WriteCsv writes connectome data in CSV format with body names as
// headers for rows/columns
func (c Connectome) WriteCsv(writer io.Writer) {

	csvWriter := csv.NewWriter(writer)
	namedBodyList := c.Neurons.SortByName()

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
			connections, preFound := c.Connectivity[namedBody1.Body]
			if preFound {
				connection, postFound := connections[namedBody2.Body]
				if postFound {
					strength = connection.Strength()
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
	csvWriter.Flush()
}

// WriteCsvFile writes connectome data into a CSV file.
func (c Connectome) WriteCsvFile(filename string) {
	file, err := os.Create(filename)
	if err != nil {
		log.Fatalf("ERROR: Failed to create connectome csv file: %s [%s]\n",
			filename, err)
	}
	c.WriteCsv(file)
	file.Close()
}

// Write every type of output file for connectome.
func (c Connectome) WriteFiles(outputDir, baseName string) {
	c.WriteMatlabFile(filepath.Join(outputDir, baseName+".m"), baseName)
	c.WriteCsvFile(filepath.Join(outputDir, baseName+".csv"))
	c.WriteNeuroptikonFile(filepath.Join(outputDir, baseName+".py"))
	c.WriteGobFile(filepath.Join(outputDir, baseName+".gob"))
	c.WriteJsonFile(filepath.Join(outputDir, baseName+".json"))
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

// MatchingNames returns a slice of body names that have prefixes matching
// the given slice of patterns
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
