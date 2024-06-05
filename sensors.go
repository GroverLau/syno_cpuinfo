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

func getDetectedChips() []*C.struct_sensors_chip_name {
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

func getFeatures(chip *C.struct_sensors_chip_name) []*C.struct_sensors_feature {
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

func getSubFeaturesNumbers(chip *C.struct_sensors_chip_name, feature *C.struct_sensors_feature) []int {
	var Numbers []int
	var count C.int = 0

	for  {
		subfeatures := C.sensors_get_all_subfeatures(chip, feature, &count)
		if subfeatures == nil {
			break
		}
		if int32(subfeatures._type) == int32(C.SENSORS_SUBFEATURE_TEMP_INPUT) {
			Numbers = append(Numbers, int(subfeatures.number))
		}

	}

	return Numbers
}

func getLabel(chip *C.struct_sensors_chip_name, feature *C.struct_sensors_feature) string {
	label := C.sensors_get_label(chip, feature)
	labelStr := C.GoString(label)
	C.free(unsafe.Pointer(label))
	return labelStr
}

func getValue(chip *C.struct_sensors_chip_name, number int) int {
	var value C.double
	C.sensors_get_value(chip, C.int(number), &value)

	return int(value)
}

func checkLabel(label string, target string) bool {
    return strings.Contains(label, target)
}


func getTempFromSensors() int{
	var temperatures int = 0

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
        return 0
    }
    defer C.sensors_cleanup()

	chips := getDetectedChips()
	for _, chip := range chips {
		features := getFeatures(chip)
		for _, feature := range features {
			numbers := getSubFeaturesNumbers(chip, feature)
			for _, number := range numbers {
				if temp := getValue(chip, number); temp > temperatures {
					temperatures = temp
				}		
			}
		}
	}
	return temperatures
}



