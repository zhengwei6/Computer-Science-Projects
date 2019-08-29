#!/usr/bin/env ruby
# This script will parse the raw temperature data by individual sensors
# Input: temperature[0-3].csv
#   Content: timestamp(string), addr(integer string), value(float string)
# Output: sensor-t-[0-3]-[0-7].csv
# 	Content: timestamp(string), value(float)

require 'csv'
require 'pathname'
require 'fileutils'

def parseTempCSV(file, indexA, targetDir)
	sensorNum = 8 # 8 different sensors in one file
	sensorFiles = Array.new(sensorNum)
	# parpare csv files for sensors
	sensorNum.times do |i|
		fName = "sensor-t-#{indexA}-#{i}.csv"
		fName = Pathname.new(targetDir) + fName
		sensorFiles[i] = CSV.open(fName, "w")
		sensorFiles[i] << ["timestamp", "value"]
	end

	# read raw temperature data
	CSV.foreach(file, :headers => true) do |row|
		# headers: timestamp, addr, value
		timestamp = row["timestamp"]
		indexB = row["addr"].to_i
		value = row["value"].to_f
		# ignore invalid sensor value
		if value < 0
			next
		end
		# check if sensor addr is valid
		if indexB >= 0 && indexB < sensorNum
			sensorFiles[indexB] << [timestamp, value]
		end
	end

	# close sensor files
	sensorNum.times do |i|
		sensorFiles[i].close()
	end
end

def startParsing(sourceDir, targetDir)
	files = Array.new
	4.times do |i|
		files << Pathname.new(sourceDir) + "temperature#{i}.csv"
	end

	files.each_index do |i|
		if files[i].exist?
			puts "Parsing #{files[i]} ..."
			parseTempCSV(files[i], i, targetDir)
		else
			puts "Failed to find #{files[i]} ..."
		end
	end
end

def main
	a = ARGV
	sourceDir = "."
	targetDir = "."
	if a.length == 0
		# use work dir
	elsif a.length == 1
		sourceDir = a[0]
		targetDir = a[0]
	elsif a.length == 2
		sourceDir = a[0]
		targetDir = a[1]
	else
		puts "do nothing ..."
		exit(-1)
	end
	if !Dir.exist?(targetDir)
		FileUtils.mkdir_p(targetDir)
	end
	startParsing(sourceDir, targetDir)
end

if __FILE__ == $0
	main
end
