package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type Tags struct {
	Tags         []Tag `json:"tags"`
	BatteryLevel int   `json:"batteryLevel"`
}

// Tag represents a measurement from a tag
type Tag struct {
	// CreateDate         time.Time `json:"createDate"`
	//HumidityOffsetDate  time.Time `json:"humidityOffsetDate"`
	AccelX          float32   `json:"accelX"`
	AccelY          float32   `json:"accelY"`
	AccelZ          float32   `json:"accelZ"`
	Connectable     bool      `json:"connectable"`
	DataFormat      int       `json:"dataFormat"`
	Humidity        float32   `json:"humidity"`
	HumidityOffset  float32   `json:"humidityOffset"`
	ID              string    `json:"id"`
	SeqNo           uint16    `json:"measurementSequenceNumber"`
	MovementCounter uint8     `json:"movementCounter"`
	Name            string    `json:"name"`
	Pressure        float32   `json:"pressure"`
	RSSI            int8      `json:"rssi"`
	Temperature     float32   `json:"temperature"`
	TxPower         int8      `json:"txPower"`
	UpdateAt        time.Time `json:"updateAt"`
	Voltage         float32   `json:"voltage"`
}

func parseRaw(raw []uint8) {
	parseAccel := func(pos int) float32 {
		return float32(int16(uint16(raw[pos])<<8|uint16(raw[pos+1]))) / 32767.
	}
	if len(raw) >= 46 {
		if raw[19] == 0x99 && raw[20] == 0x04 {
			mac := func() string {
				return fmt.Sprintf("%02X:%02X:%02X:%02X:%02X:%02X",
					raw[39], raw[40], raw[41], raw[42], raw[43], raw[44])
			}()
			t := func() float32 {
				tmp := uint16(raw[22])<<8 | uint16(raw[23])
				return float32(tmp) * .005
			}()
			h := func() float32 {
				tmp := uint16(raw[24])<<8 | uint16(raw[25])
				return float32(tmp) * .0025
			}()
			p := func() float32 {
				tmp := uint16(raw[26])<<8 | uint16(raw[27])
				return float32(uint32(tmp)+50000) / 100.0
			}()
			v := func() float32 {
				tmp := uint16(raw[34])<<8 | uint16(raw[35])
				tmp >>= 5
				return float32(uint32(tmp))/1000. + 1.6
			}()
			m := func() uint8 {
				return raw[36]
			}()
			seqno := func() uint16 {
				return uint16(raw[37])<<8 | uint16(raw[38])
			}()
			//fmt.Printf("%v %.2f %.2f %.2f %.2f %d\n", mac, t, h, p, v, m)
			tag := Tag{
				DataFormat:      5,
				UpdateAt:        time.Now(),
				ID:              mac,
				AccelX:          parseAccel(28),
				AccelY:          parseAccel(30),
				AccelZ:          parseAccel(32),
				Temperature:     t,
				Humidity:        h,
				Pressure:        p,
				Voltage:         v,
				MovementCounter: m,
				SeqNo:           seqno,
				RSSI:            func() int8 { return int8(raw[45]) }(),
			}
			// tag.ID = mac
			// tag.AccelX = parseAccel(28)
			// tag.AccelY = parseAccel(30)
			// tag.AccelZ = parseAccel(32)
			// tag.Temperature = t
			// tag.Humidity = h
			// tag.Pressure = p
			// tag.UpdateAt = time.Now()
			// tag.Voltage = v
			// tag.MovementCounter = m
			// tag.SeqNo = seqno
			// tag.RSSI = func() int8 { return int8(raw[45]) }()
			//fmt.Printf("%v %.2f %.2f %.2f %.2f %d\n", mac, t, h, p, v, m)

			tags := Tags{}
			tags.Tags = append(tags.Tags, tag)

			tagStr, _ := json.MarshalIndent(tags, "", "  ")
			fmt.Println(string(tagStr))
		}
	}
}

// scan is the background scan required for hcidump to work
func scan() {
	cmd := exec.Command("hcitool", "lescan", "--duplicates", "--passive")
	cmd.Start()
	// The application will continue running until killed. When it's kill, the
	// spawned child process will be killed too. If the application would stop
	// normally, the prcess should be killed: cmd.Process.Kill()
}

// parseDump executes hcidump and parses the output
func parseDump() {
	cmd := exec.Command("hcidump", "--raw")
	pipe, err := cmd.StdoutPipe()

	if err != nil {
		panic(err)
	}
	defer pipe.Close()
	cmd.Start()

	scanner := bufio.NewScanner(pipe)
	raw := []uint8{}
	for scanner.Scan() {
		if strings.HasPrefix(scanner.Text(), ">") {
			// > indicates the first  line of input. Thus, we can process the
			// buffered raw data
			parseRaw(raw)
			raw = nil
		}
		for _, hex := range strings.Split(scanner.Text(), " ") {
			value, err := strconv.ParseUint(hex, 16, 8)
			if err == nil {
				raw = append(raw, uint8(value))
			}
		}
	}
}

func main() {
	scan()
	parseDump()
}
