package nchc

import (
	"math"
)

func getSensorDistance(s0, s1 *SensorData) float64 {
	d := (s0.X-s1.X)*(s0.X-s1.X) + (s0.Y-s1.Y)*(s0.Y-s1.Y) + (s0.Z-s1.Z)*(s0.Z-s1.Z)
	return math.Sqrt(d)
}

func genValueAlgo0(s *SensorData, sensors []*SensorData) float64 {
	s.Value = sensors[0].Value
	dMin := getSensorDistance(s, sensors[0])
	for _, sen := range sensors {
		d := getSensorDistance(s, sen)
		if d < dMin {
			s.Value = sen.Value
			dMin = d
		}
	}

	return s.Value
}

func genValueAlgo1(s *SensorData, sensors []*SensorData) float64 {
	dp := 1.0    // distance impact
	wSur := 0.75 // material surface impact

	vSum := 0.0
	dSum := 0.0
	tValue := 0.0
	for _, sen := range sensors {
		// skip non-nchc sensor
		if sen.Type != "nchc" {
			continue
		}

		// the point is just a sensor
		if s.X == sen.X && s.Y == sen.Y && s.Z == sen.Z {
			tValue = sen.Value
			return tValue
		}

		d := getSensorDistance(s, sen)
		d = math.Pow(d, dp)
		d = 1 / d
		dSum += d
		vSum += sen.Value * d
	}
	if dSum > 0 {
		tValue = vSum / dSum
	}

	cr := []float64{
		mCubeSizeX,
		mCubeSizeY,
		mCubeSizeZ,
	}
	// handle aidc curing sensor
	for _, sen := range sensors {
		if sen.Type != "aidc" {
			continue
		}
		ds := make([]float64, 3)
		ds[0] = math.Abs(s.X - sen.X)
		ds[1] = math.Abs(s.Y - sen.Y)
		ds[2] = math.Abs(s.Z - sen.Z)

		// check if the point is inside the cube
		if ds[0] <= cr[0] && ds[1] <= cr[1] && ds[2] <= cr[2] {
			// w := (ds[0] + ds[1] + ds[2]) / (cr[0] + cr[1] + cr[2])
			// w := (ds[0]/cr[0] + ds[1]/cr[1] + ds[2]/cr[2]) / 3
			// w := (ds[0]*ds[0] + ds[1]*ds[1] + ds[2]*ds[2]) / (cr[0]*cr[0] + cr[1]*cr[1] + cr[2]*cr[2])
			w := (ds[0]*ds[0]/cr[0]/cr[0] + ds[1]*ds[1]/cr[1]/cr[1] + ds[2]*ds[2]/cr[2]/cr[2]) / 3

			vSur := sen.Value*wSur + tValue*(1.0-wSur)
			tValue = sen.Value*(1-w) + vSur*w
			return tValue
		}
	}

	return tValue
}

func genValueAlgo1FromType(s *SensorData, sensors []*SensorData, sensorType string) float64 {
	tValue := 0.0
	wSur := 0.75 // material surface impact
	dp := 1.0    // distance impact

	vSum := 0.0
	dSum := 0.0
	for _, sen := range sensors {
		// skip non-nchc sensor
		if sen.Type != "nchc" {
			continue
		}
		// the point is just a sensor
		if s.X == sen.X && s.Y == sen.Y && s.Z == sen.Z {
			tValue = sen.Value
			return tValue
		}
		d := getSensorDistance(s, sen)
		d = math.Pow(d, dp)
		d = 1 / d
		dSum += d
		vSum += sen.Value * d
	}
	if dSum > 0 {
		tValue = vSum / dSum
	}
	if sensorType == "aidc" {
		cr := []float64{
			mCubeSizeX,
			mCubeSizeY,
			mCubeSizeZ,
		}
		aidcValue := 0.0
		// handle aidc curing sensor
		for _, sen := range sensors {
			if sen.Type != "aidc" {
				continue
			}
			ds := make([]float64, 3)
			ds[0] = math.Abs(s.X - sen.X)
			ds[1] = math.Abs(s.Y - sen.Y)
			ds[2] = math.Abs(s.Z - sen.Z)

			// check if the point is inside the cube
			if ds[0] <= cr[0] && ds[1] <= cr[1] && ds[2] <= cr[2] {
				w := (ds[0]*ds[0]/cr[0]/cr[0] + ds[1]*ds[1]/cr[1]/cr[1] + ds[2]*ds[2]/cr[2]/cr[2]) / 3

				vSur := sen.Value*wSur + tValue*(1.0-wSur)
				aidcValue = sen.Value*(1-w) + vSur*w
				break
			}
		}
		tValue = aidcValue
	}

	return tValue
}
