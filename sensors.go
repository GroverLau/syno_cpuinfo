package main

/*
#cgo CFLAGS: -I/usr/local/include
#cgo LDFLAGS: -L/usr/local/lib -lsensors -lm -static
#include <stdlib.h>
#include <sensors/sensors.h>
*/
import "C"
import (
	"unsafe"
	"os"
	"log"
	"strings"
)

type SubFeatureType int32

const (
	SubFeatureTypeTempInput          SubFeatureType = C.SENSORS_SUBFEATURE_TEMP_INPUT
)

func GetDetectedChips() []*C.struct_sensors_chip_name {
	var chips []*C.struct_sensors_chip_name
	var count C.int = 0

	for {
		chip := C.sensors_get_detected_chips(nil, &count)
		if chip == nil {
			break
		}
		chips = append(chips, chip)
	}

	return chips
}

func GetFeatures(chip *C.struct_sensors_chip_name) []*C.struct_sensors_feature {
	var features []*C.struct_sensors_feature
	var count C.int = 0

	for {
		feature := C.sensors_get_features(chip, &count)
		if feature == nil {
			break
		}
		features = append(features, feature)
	}

	return features
}

func GetSubFeaturesNumbers(chip *C.struct_sensors_chip_name, feature *C.struct_sensors_feature) []int {
	var Numbers []int
	var count C.int = 0

	for  {
		subfeatures := C.sensors_get_all_subfeatures(chip, feature, &count)
		if subfeatures == nil {
			break
		}
		if SubFeatureType(subfeatures._type) == SubFeatureTypeTempInput {
			Numbers = append(Numbers, int(subfeatures.number))
		}

	}

	return Numbers
}

func GetLabel(chip *C.struct_sensors_chip_name, feature *C.struct_sensors_feature) string {
	clabel := C.sensors_get_label(chip, feature)
	golabel := C.GoString(clabel)
	C.free(unsafe.Pointer(clabel))
	return golabel
}

func GetValue(chip *C.struct_sensors_chip_name, number int) int {
	var value C.double
	C.sensors_get_value(chip, C.int(number), &value)

	return int(value)
}

func checkLabel(label string, target string) bool {
    return strings.Contains(label, target)
}


func getTempFromSensors() []int{
	var temperatures []int

    var fp *C.FILE
    if _, err := os.Stat(sensorsConfigPath); os.IsNotExist(err) {
        fp = nil
    } else {
        filename := C.CString(sensorsConfigPath)
        defer C.free(unsafe.Pointer(filename))

        mode := C.CString("r")
        defer C.free(unsafe.Pointer(mode))

        fp = C.fopen(filename, mode)
        if fp == nil {
            log.Fatal("Failed to open configuration file")
        }
        defer C.fclose(fp)
    }

    if res := C.sensors_init(fp); res != 0 {
        log.Println("Failed to initialize sensors")
        return []int{0}
    }
    defer C.sensors_cleanup()

	chips := GetDetectedChips()
	for _, chip := range chips {
		features := GetFeatures(chip)
		for _, feature := range features {
			numbers := GetSubFeaturesNumbers(chip, feature)
			for _, number := range numbers {
				if checkLabel(GetLabel(chip,feature),"Core "){
					temp :=GetValue(chip,number)
					temperatures = append(temperatures, temp)
				}			
			}
		}
	}
	return temperatures
}

func findMaxTemperature(temperatures []int) int {
	if len(temperatures) == 0 {
		return 0
	}

	maxTemp := temperatures[0]
	for _, temp := range temperatures {
		if temp > maxTemp {
			maxTemp = temp
		}
	}
	return maxTemp
}


