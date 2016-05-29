// Copyright (c) 2016, Antonio Zanardo <zanardo@gmail.com>
// All rights reserved.
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are met:
// 
//  * Redistributions of source code must retain the above copyright
//    notice, this list of conditions and the following disclaimer.
//  * Redistributions in binary form must reproduce the above copyright
//    notice, this list of conditions and the following disclaimer in the
//    documentation and/or other materials provided with the distribution.
// 
// THIS SOFTWARE IS PROVIDED BY THE AUTHOR AND CONTRIBUTORS ``AS IS'' AND ANY
// EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE IMPLIED
// WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE ARE
// DISCLAIMED. IN NO EVENT SHALL THE AUTHOR AND CONTRIBUTORS BE LIABLE FOR ANY
// DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR CONSEQUENTIAL DAMAGES
// (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF SUBSTITUTE GOODS OR SERVICES;
// LOSS OF USE, DATA, OR PROFITS; OR BUSINESS INTERRUPTION) HOWEVER CAUSED AND
// ON ANY THEORY OF LIABILITY, WHETHER IN CONTRACT, STRICT LIABILITY, OR TORT
// (INCLUDING NEGLIGENCE OR OTHERWISE) ARISING IN ANY WAY OUT OF THE USE OF THIS
// SOFTWARE, EVEN IF ADVISED OF THE POSSIBILITY OF SUCH DAMAGE.

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	Version     = "0.1"
	MinFreqFile = "/sys/devices/system/cpu/cpu0/cpufreq/cpuinfo_min_freq"
	MaxFreqFile = "/sys/devices/system/cpu/cpu0/cpufreq/cpuinfo_max_freq"
)

var (
	PossibleTempFiles = []string{
		"/sys/class/thermal/thermal_zone0/temp",
		"/sys/class/thermal/thermal_zone1/temp",
		"/sys/class/thermal/thermal_zone2/temp",
		"/sys/class/hwmon/hwmon0/temp1_input",
		"/sys/class/hwmon/hwmon1/temp1_input",
		"/sys/class/hwmon/hwmon2/temp1_input",
		"/sys/class/hwmon/hwmon0/device/temp1_input",
		"/sys/class/hwmon/hwmon1/device/temp1_input",
		"/sys/class/hwmon/hwmon2/device/temp1_input",
	}
)

func parseFreqFile(path string) int64 {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal("error opening ", path)
	}
	freqstr := strings.TrimSpace(string(data))
	freq, err := strconv.ParseInt(freqstr, 10, 64)
	if err != nil {
		log.Fatal("error parsing frequency in ", path, ": ", err)
	}
	return freq
}

func parseTempFile(path string) (int64, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return 0, err
	} else {
		data, err := ioutil.ReadFile(path)
		if err != nil {
			log.Fatal("error opening ", path)
		}
		tempstr := strings.TrimSpace(string(data))
		temp, err := strconv.ParseInt(tempstr, 10, 64)
		if err != nil {
			log.Fatal("error parsing temperature in ", path, ": ", err)
		}
		return temp, nil
	}
}

func collectMinFreq() int64 {
	return parseFreqFile(MinFreqFile)
}

func collectMaxFreq() int64 {
	return parseFreqFile(MaxFreqFile)
}

func collectTemp() float64 {
	for _, tempFile := range PossibleTempFiles {
		temp, err := parseTempFile(tempFile)
		if err == nil {
			return float64(temp) / float64(1000)
		}
	}
	log.Fatal("error collecting temperature from all known files")
	return 0
}

func setFrequency(freq int64, num_cpus int) {
	for cpu := 0; cpu < num_cpus; cpu++ {
		filePath := fmt.Sprintf("/sys/devices/system/cpu/cpu%d/cpufreq/scaling_max_freq", cpu)
		err := ioutil.WriteFile(filePath, []byte(fmt.Sprintf("%d\n", freq)), 0755)
		if err != nil {
			log.Fatal("error setting frequency ", freq, " to file ", filePath, ": ", err)
		}
	}
	log.Printf("cpu frequency set to %d", freq)
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("usage:", os.Args[0], "<max temp>")
		fmt.Println("ex:", os.Args[0], "70")
		os.Exit(1)
	}
	max_temp, err := strconv.ParseFloat(os.Args[1], 64)
	if err != nil {
		fmt.Println("error parsing", os.Args[1], "to float")
		os.Exit(1)
	}
	log.Print("starting throttle cpu temp - version ", Version)
	log.Print("maximum temperature: ", max_temp)

	min_freq := collectMinFreq()
	log.Print("minimum frequency: ", min_freq)
	max_freq := collectMaxFreq()
	log.Print("maximum frequency: ", max_freq)

	num_cpus := runtime.NumCPU()
	log.Print("number of cores: ", num_cpus)

	cur_freq := max_freq
	setFrequency(cur_freq, num_cpus)
	var step int64 = 100000

	for {
		temp := collectTemp()
		if temp > max_temp && cur_freq > min_freq {
			cur_freq -= step
			if cur_freq < min_freq {
				cur_freq = min_freq
			}
			setFrequency(cur_freq, num_cpus)
		} else if temp < (max_temp-5.0) && cur_freq < max_freq {
			cur_freq += step
			if cur_freq > max_freq {
				cur_freq = max_freq
			}
			setFrequency(cur_freq, num_cpus)
		}
		time.Sleep(time.Second * 3)
	}
}
