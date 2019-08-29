#!/usr/bin/env ruby
# This script will transfer temperature degree
# Input: temperature[0-3].csv
#   Content: timestamp(string), addr(integer string), value(float string)
# Output: f-temperature[0-3].csv
#   Content: timestamp(string), addr(integer string), value(float string)


require 'csv'
require 'pathname'
require 'fileutils'

def tempF2C(t)
  return (t.to_f - 32) * 5 / 9
end

def tempC2F(t)
  return t.to_f * 9 / 5 + 32
end

def parseTempCSV(file, indexA, targetDir)
  # prepare new file
  fName = "f-temperature#{indexA}.csv"
  fName = Pathname.new(targetDir) + fName
  csv = CSV.open(fName, "w")
  csv << ["timestamp", "addr", "value"]

	# read raw temperature data
	CSV.foreach(file, :headers => true) do |row|
		# headers: timestamp, addr, value
		timestamp = row["timestamp"]
		indexB = row["addr"].to_i
		value = row["value"].to_f

    # transfer value
    value = tempC2F(value)

    csv << [timestamp, indexB, value]
	end
  csv.close()
end

def startParsing(sourceDir, targetDir)
	files = Array.new
	4.times do |i|
		files << Pathname.new(sourceDir) + "temperature#{i}.csv"
	end

	files.each_index do |i|
		puts "Transfering #{files[i]} ..."
		parseTempCSV(files[i], i, targetDir)
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
