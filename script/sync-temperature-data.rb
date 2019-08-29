#!/usr/bin/env ruby
# This script will sync timestamp of temperature data for sensors
# Input: sensor-t-[0-3]-[0-7].csv
# 	Content: timestamp(string), value(float)
# Output: sync-sensor-t-[0-3]-[0-7].csv
# 	Content: timestamp(integer), value(float)

require 'csv'
require 'time'
require 'pathname'
require 'fileutils'

def syncTimestamp(file, file2)
	newFile = file2
	startTime = 0
	endTime = 0
	CSV.open(newFile, "w") do |csv|
		prevTime = 0
		prevValue = 0.0
		csv << ["timestamp", "value"]
		timeA = 0
		valueA = 0.0
		timeB = 0
		valueB = 0.0
		targetTime = 0
		targetValue = 0.0
		CSV.foreach(file, :headers => true) do |row|
			# headers: timestamp, value
			timestamp = Time.parse(row["timestamp"]).to_i
			value = row["value"].to_f
			# skip abnormal value
			if value <= 0 || value > 700
				next
			end
			# sync value by time with interpolation every 10 seconds
			if startTime == 0
				# retrieve first record
				timeA = timestamp
				valueA = value
				timeB = timestamp
				valueB = value
				startTime = timestamp + (10 - timestamp % 10) # record first sync time
				targetTime = startTime
				next
			end

			if timestamp > targetTime
				# update reference range
				timeA = timeB
				valueA = valueB
				timeB = timestamp
				valueB = value
			else
				next
			end

			while timeB > targetTime
				d = timeB - timeA # note, consider to discard data if d is too large
				targetValue = ((targetTime - timeA) * valueB + (timeB - targetTime) * valueA) / d
				csv << [targetTime, targetValue]
				endTime = targetTime
				targetTime += 10
			end
		end
	end
	if startTime > 0 && endTime > 0
		puts " >> #{Time.at(startTime)} ~ #{Time.at(endTime)}"
	else
		startTime = 0
		endTime = 0
		puts " no data"
	end
	return startTime, endTime
end

def syncSensorFiles(sourceDir, targetDir)
	hubNum = 4 # total 4 hubs
	sensorNum = 8 # 8 different sensors in one hub
	minStartTime = 0
	maxEndTime = 0
	hubNum.times do |indexA|
		sensorNum.times do |indexB|
			fName = "sensor-t-#{indexA}-#{indexB}.csv"
			fName2 = "sync-" + fName
			fName = Pathname.new(sourceDir) + fName
			fName2 = Pathname.new(targetDir) + fName2
			if fName.exist?
				print "Parsing #{fName} ..."
			else
				puts "Failed to find #{fName} ..."
				return # exit if one file can not be found
			end
			startTime, endTime = syncTimestamp(fName, fName2)
			if startTime > 0
				if minStartTime == 0
					minStartTime = startTime
				elsif startTime < minStartTime
					minStartTime = startTime
				end
			end
			if endTime > 0
				if maxEndTime == 0
					maxEndTime = endTime
				elsif endTime > maxEndTime
					maxEndTime = endTime
				end
			end
		end
	end
	puts "Aligned: #{Time.at(minStartTime)} ~ #{Time.at(maxEndTime)}"

	# create a new file to store all sync values
	outputName = "sync-output-plot.csv"
	outputName = Pathname.new(targetDir) + outputName
	puts "Each sensor should contribute #{(maxEndTime - minStartTime)/10 + 1} records"
	syncData = Hash.new
	hubNum.times do |indexA|
		sensorNum.times do |indexB|
			fName = "sync-sensor-t-#{indexA}-#{indexB}.csv"
			fName = Pathname.new(targetDir) + fName
			print "Loading #{fName} ..."
			newCol = Hash.new
			workingTimestamp = 0
			CSV.foreach(fName, :headers => true) do |row|
				timestamp = row["timestamp"].to_i
				next if timestamp < minStartTime
				break if timestamp > maxEndTime
				if workingTimestamp == 0 # append head data
					workingTimestamp = minStartTime
					while workingTimestamp < timestamp
						newCol[workingTimestamp] = 9999.0
						workingTimestamp += 10
					end
				end
				value = row["value"].to_f
				newCol[timestamp] = value
				workingTimestamp = timestamp
			end
			if workingTimestamp != 0
				workingTimestamp += 10
			else # no data
				workingTimestamp = minStartTime
			end
			while workingTimestamp <= maxEndTime # append tail data
				newCol[workingTimestamp] = 9999.0
				workingTimestamp += 10
			end
			puts " >> #{newCol.length} records"
			if newCol.length > 0
				syncData["#{indexA}-#{indexB}"] = newCol
			end
		end
	end

	keys = syncData.keys
	header = ["timestamp"].concat(keys)
	CSV.open(outputName, "w") do |csv|
		csv << header
		i = minStartTime
		while i <= maxEndTime
			data = [i]
			keys.each do |k|
				data << syncData[k][i]
			end
			csv << data
			i += 10
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
	syncSensorFiles(sourceDir, targetDir)
end

if __FILE__ == $0
	main
end
