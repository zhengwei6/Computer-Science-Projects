#!/usr/bin/env ruby
# This script will generate 3D vtk file for given temperature

require 'csv'
require 'time'

Struct.new("Layout", :algo, :oven, :dp, :scale, :xCenter, :yCenter, :zCenter, :radius, :xGrid, :yGrid, :zGrid)
Struct.new("Sensor", :desc, :x, :y, :z, :value)

# check if the point is located inside the cabin
def ifInternal(x, y, z, layout)
	# TODO: update this range to fit real depth
	# if x < -62 || x > 58
	# 	return false
	# end
	# check if the point is in the circle
	if y * y + z * z > layout.radius * layout.radius
		return false
	end
	return true
end

# for testing only
def genTestValue(x, y, z, layout)
	if ifInternal(x, y, z, layout)
		r2 = (layout.radius * layout.radius).to_f
		v = (1.0 - y * y / r2 - z * z / r2) * 100
		return v
	end
	return 0
end

# Refer to the closest sensor
def genValueAlgo0(x, y, z, layout, sensors)
	if ifInternal(x, y, z, layout)
		# used the cloest sensor value
		v = 0
		minDistance = nil
		sensors.each_value do |sen|
			d = (x-sen.x)*(x-sen.x) + (y-sen.y)*(y-sen.y) + (z-sen.z)*(z-sen.z)
			d = d ** (0.5 * layout.dp)
			if minDistance == nil
				minDistance = d
				v = sen.value
				next
			end
			if d < minDistance
				minDistance = d
				v = sen.value
			end
		end
		return v
	end
	return 0
end

# Refer to all sensors
def genValueAlgo1(x, y, z, layout, sensors)
	if ifInternal(x, y, z, layout)
		vSum = 0
		dSum = 0
		sensors.each_value do |sen|
			if x == sen.x && y == sen.y && z == sen.z
				return sen.value
			end
			d = (x-sen.x)*(x-sen.x) + (y-sen.y)*(y-sen.y) + (z-sen.z)*(z-sen.z)
			d = d ** (0.5 * layout.dp)
			d = 1.0 / d.to_f
			dSum += d
			vSum += sen.value * d
		end
		v = vSum / dSum if dSum > 0
		if v.nan?
			puts "Invalid value at [#{x},#{y},#{z}] : #{v}"
			return 0
		end
		return v
	end
	return 0
end

def genValue(x, y, z, layout, sensors)
	if layout.algo == 1
		return genValueAlgo1(x, y, z, layout, sensors)
	end
	return genValueAlgo0(x, y, z, layout, sensors)
end

# generate VTK header of structured points based on x, y, z dimensions
def genVTKHeader(x, y, z)
	header = "# vtk DataFile Version 1.0\n"
	header += "test\n"
	header += "ASCII\n"
	header += "DATASET STRUCTURED_POINTS\n"
	header += "DIMENSIONS #{x} #{y} #{z}\n"
	header += "ORIGIN 0.0 0.0 0.0\n"
	header += "SPACING 1.0 1.0 1.0\n"

	points = x * y * z
	header += "\nPOINT_DATA #{points}\n\n"
end

# generate VTK file of structured points
def genVTKFile(fileName, layout, sensors)
	xGrid = layout.xGrid
	yGrid = layout.yGrid
	zGrid = layout.zGrid
	header = genVTKHeader(xGrid*2+1, yGrid*2+1, zGrid*2+1)
	File.open(fileName, "w") do |f|
		f << header
		f << "SCALARS temperature float\n"
		f << "LOOKUP_TABLE default\n"
		(-zGrid..zGrid).each do |z|
		  (-yGrid..yGrid).each do |y|
		    (-xGrid..xGrid).each do |x|
			    v = genValue(x, y, z, layout, sensors)
			    f << v << "\n"
		    end
		  end
		end
	end
end

# get initial sensor status
def getSensors(layout)
	sensors = Hash.new
	if layout.oven == "a"
		# Section 1
		sensors["1-0"] = Struct::Sensor.new("C26", 184, -85, 277, -1)
		sensors["1-1"] = Struct::Sensor.new("C27", 184, 0, 365, -1)
		sensors["2-4"] = Struct::Sensor.new("C56", 184, 85, 277, -1)

		# Section 2
		sensors["0-6"] = Struct::Sensor.new("C19", 334, -85, 277, -1)
		sensors["0-7"] = Struct::Sensor.new("C20", 334, 0, 365, -1)
		sensors["2-3"] = Struct::Sensor.new("C48", 334, 85, 277, -1)

		# Section 3
		sensors["0-4"] = Struct::Sensor.new("C17", 484, -85, 277, -1)
		sensors["0-5"] = Struct::Sensor.new("C18", 484, 0, 365, -1)
		sensors["2-2"] = Struct::Sensor.new("C47", 484, 85, 277, -1)

		# Section 4
		sensors["0-2"] = Struct::Sensor.new("C11", 634, -85, 277, -1)
		sensors["0-3"] = Struct::Sensor.new("C12", 634, 0, 365, -1)
		sensors["2-1"] = Struct::Sensor.new("C41", 634, 85, 277, -1)

		# Section 5
		sensors["0-0"] = Struct::Sensor.new("C9", 784, -85, 277, -1)
		sensors["0-1"] = Struct::Sensor.new("C10", 784, 0, 365, -1)
		sensors["2-0"] = Struct::Sensor.new("C40", 784, 85, 277, -1)

	  # Rack in
		sensors["1-6"] = Struct::Sensor.new("C28", 20.5, 0, 88, -1)
		sensors["3-1"] = Struct::Sensor.new("C59", 20.5, 0, 188, -1)
		sensors["1-7"] = Struct::Sensor.new("C29", 155.5, 0, 88, -1)
		sensors["3-0"] = Struct::Sensor.new("C58", 155.5, 0, 188, -1)
		sensors["1-5"] = Struct::Sensor.new("C22", 323.5, 0, 88, -1)
		sensors["2-7"] = Struct::Sensor.new("C50", 323.5, 0, 188, -1)

		# Rack Out
		# sensors["1-3"] = Struct::Sensor.new("C13", ?, 0, 88, -1)
		# sensors["3-2"] = Struct::Sensor.new("C44", ?, 0, 188, -1)
		sensors["1-4"] = Struct::Sensor.new("C14", 654.5, 0, 88, -1)
		sensors["2-6"] = Struct::Sensor.new("C42", 654.5, 0, 188, -1)
		sensors["1-2"] = Struct::Sensor.new("C6", 808.5, 0, 88, -1)
		sensors["2-5"] = Struct::Sensor.new("C32", 808.5, 0, 188, -1)

	else
		# oven B
		# hub 0
		sensors["0-0"] = Struct::Sensor.new("C10", 784, -85, 277, -1)
		sensors["0-1"] = Struct::Sensor.new("C11", 634, -85, 277, -1)
		sensors["0-2"] = Struct::Sensor.new("C12", 784, 0, 365, -1)
		sensors["0-3"] = Struct::Sensor.new("C16", 634, 0, 365, -1)
		sensors["0-4"] = Struct::Sensor.new("C17", 484, -85, 277, -1)
		sensors["0-5"] = Struct::Sensor.new("C18", 484, 0, 365, -1)
		sensors["0-6"] = Struct::Sensor.new("C19", 334, -85, 277, -1)
		sensors["0-7"] = Struct::Sensor.new("C20", 334, 0, 365, -1)
		# hub 1
		sensors["1-0"] = Struct::Sensor.new("C25", 184, -85, 277, -1)
		sensors["1-1"] = Struct::Sensor.new("C26", 184, 0, 365, -1)
		# hub 2
		sensors["2-0"] = Struct::Sensor.new("C43", 784, 85, 277, -1)
		sensors["2-1"] = Struct::Sensor.new("C45", 634, 85, 277, -1)
		sensors["2-2"] = Struct::Sensor.new("C48", 484, 85, 277, -1)
		sensors["2-3"] = Struct::Sensor.new("C49", 334, 85, 277, -1)
		sensors["2-4"] = Struct::Sensor.new("C57", 184, 85, 277, -1)

		# Rack in
		sensors["1-6"] = Struct::Sensor.new("C28", 20.5, 0, 88, -1)
		sensors["3-1"] = Struct::Sensor.new("C59", 20.5, 0, 188, -1)
		sensors["1-7"] = Struct::Sensor.new("C29", 155.5, 0, 88, -1)
		sensors["3-0"] = Struct::Sensor.new("C58", 155.5, 0, 188, -1)
		sensors["1-5"] = Struct::Sensor.new("C22", 323.5, 0, 88, -1)
		sensors["2-7"] = Struct::Sensor.new("C50", 323.5, 0, 188, -1)

		# Rack Out
		# sensors["1-3"] = Struct::Sensor.new("C13", ?, 0, 88, -1)
		# sensors["3-2"] = Struct::Sensor.new("C44", ?, 0, 188, -1)
		sensors["1-4"] = Struct::Sensor.new("C14", 654.5, 0, 88, -1)
		sensors["2-6"] = Struct::Sensor.new("C42", 654.5, 0, 188, -1)
		sensors["1-2"] = Struct::Sensor.new("C6", 808.5, 0, 88, -1)
		sensors["2-5"] = Struct::Sensor.new("C32", 808.5, 0, 188, -1)
	end

	# update coordinate
	sensors.each do |key, sen|
		scale = layout.scale
		sensors[key].x = (sen.x.to_f / scale).to_i - layout.xCenter
		sensors[key].y = (sen.y.to_f / scale).to_i - layout.yCenter
		sensors[key].z = (sen.z.to_f / scale).to_i - layout.zCenter
	end

	return sensors
end

def showSensorValue(sensors)
	sensors.each_value do |sen|
		puts "#{sen.desc} [#{sen.x},#{sen.y},#{sen.z}] => #{sen.value}"
	end
end

# get start timestamp from a sync file
def getStartTime(file)
	CSV.foreach(file, :headers => true) do |row|
		# timestamp (integer), value (float)
		return row["timestamp"].to_i
	end
	return 0
end

# get sensor value from a sync file
def getValueByTime(file, timestamp)
	CSV.foreach(file, :headers => true) do |row|
		# timestamp (integer), value (float)
		if row["timestamp"].to_i == timestamp
			return row["value"].to_f
		end
	end
	return 0
end

# get max start timestamp from sync files
def getMaxStartTime()
	hubNum = 4 # total 4 hubs
	sensorNum = 8 # 8 different sensors in one hub
	maxStartTime = 0
	hubNum.times do |indexA|
		sensorNum.times do |indexB|
			fName = "sync-sensor-t-#{indexA}-#{indexB}.csv"
			startTime = getStartTime(fName)
			if startTime > 0
				if maxStartTime == 0
					maxStartTime = startTime
				elsif startTime > maxStartTime
					maxStartTime = startTime
				end
			end
		end
	end
	return maxStartTime
end

# get sensor values from sync files
def getAllValueByTime(timestamp)
	values = Hash.new
	hubNum = 4 # total 4 hubs
	sensorNum = 8 # 8 different sensors in one hub
	maxStartTime = 0
	hubNum.times do |indexA|
		sensorNum.times do |indexB|
			fName = "sync-sensor-t-#{indexA}-#{indexB}.csv"
			k = "#{indexA}-#{indexB}"
			v = getValueByTime(fName, timestamp)
			if v > 0
				values[k] = v
			end
		end
	end
	return values
end

def updateSensorValue(sensors, values)
	newSensors = Hash.new
	sensors.each do |key, sen|
		if values[key] && values[key] > 0
			newSensors[key] = sen
			newSensors[key].value = values[key]
		end
	end
	return newSensors
end

def example_run(layout)
	sensors = getSensors(layout)
	# read real sensor value to sensors struct
	values = {
		"1-0" => 140.0, "1-1" => 170.0, "2-4" => 150.0,
		"0-6" => 150.0, "0-7" => 140.0, "2-3" => 130.0,
		"0-4" => 140.0, "0-5" => 170.0, "2-2" => 150.0,
		"0-1" => 150.0, "0-3" => 140.0, "2-1" => 130.0,
		"0-0" => 130.0, "0-2" => 150.0, "2-0" => 140.0,
	}
	newSensors = updateSensorValue(sensors, values)
	showSensorValue(newSensors)
	layout.algo = 1
	genVTKFile("example1-temperature.vtk", layout, newSensors)
	layout.algo = 0
	genVTKFile("example0-temperature.vtk", layout, newSensors)
end

def genSingleFile(layout)
	sensors = getSensors(layout)

	t = getMaxStartTime()
	puts "Start time => #{Time.at(t)}"
	values = getAllValueByTime(t)

	newSensors = updateSensorValue(sensors, values)
	showSensorValue(sensors)
	genVTKFile("temperature.vtk", layout, sensors)
end

def genMultipleFiles(layout)
	sensors = getSensors(layout)

	t = getMaxStartTime()
	count = 0

	while count < 3
		values = getAllValueByTime(t)
		if values.size == 0
			break
		end

		puts "Prasing values at #{Time.at(t)} ..."
		newSensors = updateSensorValue(sensors, values)
		showSensorValue(newSensors)
		tag = Time.at(t).strftime("%y%m%d-%H%M%S")
		puts tag
		layout.algo = 0
		genVTKFile("temperature.#{tag}.0.vtk", layout, newSensors)
		layout.algo = 1
		genVTKFile("temperature.#{tag}.1.vtk", layout, newSensors)
		t += 60 * 60 # in seconds
		count += 1
	end

end

def genFiles(oven)
	# init config
	algo = 0
	# oven = "b" # oven, a, b, or c
	dp = 1.0 # distance parameter
	scale = 10.0
	xCenter = 48
	yCenter = 0
	zCenter = 18
	radius = 18
	xGrid = 30
	yGrid = 18
	zGrid = 18
	layout = Struct::Layout.new(algo, oven, dp, scale, xCenter, yCenter, zCenter, radius, xGrid, yGrid, zGrid)
	# example_run(layout)
	# genSingleFile(layout)
	genMultipleFiles(layout)
end

def main
	a = ARGV
	if a.length == 1
		genFiles("b") if a[0] == "b" || a[0] == "B"
		genFiles("a") if a[0] == "a" || a[0] == "A"
	else
		puts "do nothing ..."
	end
end

if __FILE__ == $0
	main
end
