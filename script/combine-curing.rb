#!/usr/bin/env ruby
# Usage: combine-curing.rb tmp_curing-data.csv

require 'csv'
require 'time'
require 'pathname'
require 'fileutils'

def loadCuringCSV(cFile, sFile, nFile)
  csv = CSV.read(cFile)
  if csv.length < 2
    puts "error: no curing data"
    return
  end
  sensors = CSV.read(sFile)
  sIndex = 2 # interpolation will take current value and previous value, so index starts from 2 where 0 indicates header
  if sensors.length < 3
    puts "error: no sensor data"
    return
  end
  sNum = sensors[0].length - 1 # number of sensors

  newCSV = CSV.open(nFile, "w")
  # csv[0].pop  pop to remove the last empty element
  newCSV << csv[0].concat(sensors[0][1..-1]) # ignore timestamp

  csv.each_with_index do |v,i|
    # if i == 0 || i % 10 != 1 # i == 0 indicates headers
    if i == 0
      next
    end
    # timestamp = Time.parse(v[0] + " " + v[1])
    # t = timestamp.to_i
    t = v[0].to_i
    while sIndex < sensors.length && sensors[sIndex][0].to_i < t
      sIndex += 1
    end
    if sIndex < sensors.length && sensors[sIndex][0].to_i >= t # try to find correct time period
      t0 = sensors[sIndex-1][0].to_i
      t1 = sensors[sIndex][0].to_i
      a = Array.new
      # a << t # timestamp
      for j in 1..sNum # 0 is timestamp, so index starts from 1
        s0 = sensors[sIndex-1][j].to_f
        s1 = sensors[sIndex][j].to_f
        # tag = sensors[0][j]
        sTarget = 0
        if s0 <= 0 || s0 > 700 || s1 <= 0 || s1 > 700
          sTarget = 9999.0
        else
          sTarget = (s0 * (t1 - t) + s1 * (t - t0)) / (t1 - t0)
        end
        # a << tempC2F(sTarget)
        a << sTarget
      end
      # print "."
      # puts "#{i} XX T: #{timestamp} => #{t % 100000} [#{t0 % 100000}, #{t1 % 100000}]"
      # v.pop # skip the last empty element
      newCSV << v.concat(a)
    end
    # plotData[0] << Time.parse(v[0] + " " + v[1]).strftime("%Y-%m-%dT%H:%M:%S")
  end
  newCSV.close()
end

def testResult(nFile)
  count = 0
  puts "--------------------"
  CSV.foreach(nFile, :headers => true) do |row|
    # puts row.headers if count == 0
    # puts row if count <= 3
    if count == 0
      row.headers.each do |k|
        puts "#{k} => #{row[k]}"
      end
      puts "===================="
    end
    count += 1
  end
end

def syncSensorFiles(sourceDir, targetDir)
  iFile = "curing-data.csv"
  iFile = Pathname.new(sourceDir) + iFile
  sFile = "sync-output-plot.csv"
  sFile = Pathname.new(sourceDir) + sFile
  nFile = "sync-curing-plot.csv"
  nFile = Pathname.new(targetDir) + nFile

  if iFile.exist? && sFile.exist?
    puts "Combing #{iFile} and #{sFile} ..."
    loadCuringCSV(iFile, sFile, nFile)
    testResult(nFile)
  elsif !iFile.exist?
    puts "Failed to find #{iFile} ..."
  elsif !sFile.exist?
    puts "Failed to find #{sFile} ..."
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
