#!/usr/bin/env ruby
# Usage: parse-curing-data.rb <source directory> <target directory>

require 'csv'
require 'time'
require 'pathname'
require 'fileutils'

def startParsing(sourceDir, targetDir)
  Dir.new(sourceDir).each do |x|
    v = /\ACalc\p{ASCII}*\.csv$/ === x
    next if !v
    puts "Parsing #{x}"
    purgeCuringCSV(x, sourceDir, targetDir)
    convertCuringData(x, targetDir, targetDir)
    break # process one file only
  end
end

def purgeCuringCSV(inName, sourceDir, tempDir)
  outName = "tmp_" + inName
  outName = Pathname.new(tempDir) + outName
  inName = Pathname.new(sourceDir) + inName

  # remove unuseful lines to correct csv
  File.open(outName, 'w') do |out_file|
    File.foreach(inName).with_index do |line,line_number|
      # remove line 0, 1, 3, line 2 is header
      if line_number < 2 || line_number == 3
        # puts "Remove #{line_number} => #{line}"
        next
      end
      # write new data
      out_file.puts line
    end
  end
end

def convertCuringData(inName, tempDir, targetDir)
  # outName = "parsed_" + inName
  outName = "curing-data.csv"
  outName = Pathname.new(targetDir) + outName
  inName = "tmp_" + inName
  inName = Pathname.new(tempDir) + inName

  raw_headers = []
  headers = []
  headers_index = []
  indexFecha = -1
  indexHora = -1

  pre_headers = ["Seg", "TSP", "HOT", "COLD", "ASP", "AMV", "TTA1", "TTA2", "PSP", "PMV", "VSPA", "VMVA"]
  pre_regs = [/\AT\d*$/, /\AV\d*$/]

  count = -1
  puts "#{inName} ==> #{outName}"
  CSV.open(outName, "wb") do |csv|
    CSV.foreach(inName) do |row|
      count += 1
      if count == 0
        raw_headers = row
        puts "raw_headers: #{raw_headers}"
        if raw_headers.include?("Fecha") &&  raw_headers.include?("Hora") # check if timestamp exists
          indexFecha = raw_headers.index("Fecha")
          indexHora = raw_headers.index("Hora")
          raw_headers.each do |hx|
            addToHeader = false
            if pre_headers.include?(hx) # check pre-defined headers
              addToHeader = true
            else
              pre_regs.each { |rr| # check dynamic sensors
                if rr === hx
                  addToHeader = true
                  break
                end
              }
            end
            if addToHeader # add this header for output
              headers << hx
              headers_index << raw_headers.index(hx)
            end
          end
        else # now timestamp field
          puts "error: no timestamp fields: Fecha & Hora"
          break
        end
      end
      tString = row[indexFecha] + " " + row[indexHora]
      timestamp = 0
      if count == 0
        tString = "DateTime"
        timestamp = "timestamp"
      else
        # handle date format if necessary
        timestamp = Time.parse(tString).to_i
      end
      tmpRow = [timestamp, tString]
      headers_index.each do |hi|
        tmpRow << row[hi].strip
      end
      puts "#{count} <#{tmpRow.length}> #{tmpRow}" if count < 2
      csv << tmpRow
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
