import sys
import glob
import time
import os
import numpy as np
import pandas as pd
import datetime
from matplotlib import pyplot as plt
from scipy import signal
from sklearn.externals import joblib
from sklearn.decomposition import PCA

def select_frequecy(recipe,axis,datatype):
    filePath = os.path.join('.', 'vibrationData', 'Select_Frequency.csv')
    df = pd.read_csv(filePath,names = ['recipe','type','axis','frequency','mean','std'])
    find = df.loc[df['recipe'] == recipe]
    find = find.loc[find['axis'] == axis]
    find = find.loc[find['type'] == datatype]
    find = find.reset_index(drop=True)
    if find.shape[0] == 1:
        return find['frequency'][0].split(';') 
    else:
        return []
def computeZscore(recipe,axis,datatype):
    filePath = os.path.join('.', 'vibrationData', 'Select_Frequency.csv')
    df = pd.read_csv(filePath,names = ['recipe','type', 'axis','frequency','mean','std'])
    find = df.loc[df['recipe'] == recipe]
    find = find.loc[find['axis'] == axis]
    find = find.loc[find['type'] == datatype]
    find = find.reset_index(drop=True)
    if find.shape[0] == 1:
        return find['mean'][0],find['std'][0]
    else:
        return 0,0
def findVibrationFile(vibrationPath, startTime, endTime, sensorLabel):
    """
    find necessary vibration data paths list 
    """
    # find vibration data index
    startTime = startTime[0:15] # 2019-05-02_1120
    endTime   = endTime[0:15]
    files = []
    
    # if the vibration data is cross day
    if not(startTime[0:10] == endTime[0:10]) :
        vibrationFileNameDay1 = startTime[0:10]
        vibrationFileNameDay2 = endTime[0:10]
        
        vibrationFilePathDay1 = os.path.join('.',vibrationPath,vibrationFileNameDay1,'*'+sensorLabel+ '.csv')
        vibrationFilePathDay2 = os.path.join('.',vibrationPath,vibrationFileNameDay2,'*'+sensorLabel + '.csv')
        try:
            filesDay1 = glob.glob(vibrationFilePathDay1)
            filesDay1.sort()
            startAppend = 0
            for i, filename in enumerate(filesDay1):
                if startTime in filename:
                    startAppend = 1
                if startAppend == 1:
                    files.append(filename)
            filesDay2 = glob.glob(vibrationFilePathDay2)
            filesDay2.sort()
            startAppend = 1

            for i, filename in enumerate(filesDay2):
                if endTime in filename:
                    startAppend = 0
                if startAppend == 1:
                    files.append(filename)
            
        except Exception as e :
            print(e)
            sys.exit(1)
    else:
        vibrationFileName = startTime[0:10]
        vibrationFilePath = os.path.join('.',vibrationPath,vibrationFileName,'*'+sensorLabel+ '.csv')
        try:
            files = glob.glob(vibrationFilePath)
            files.sort()
            start_index = 0
            end_index = 0
            for i, filename in enumerate(files):
                if startTime in filename:
                    start_index = i
                elif endTime in filename:
                    end_index = i
            if len(files) == 0 or end_index <= start_index:
                print('load ' + vibrationFilePath + ' fail!')
                sys.exit(1)
        except Exception as e :
            print('load ' + vibrationFilePath + ' fail!')
            sys.exit(1)
        files = files[start_index:end_index]
    
    if len(files) == 0:
        print('load fail!')
        sys.exit(1)
    return files

def main():
    start_index = 0
    end_index = 0
    cross     = 0
    savePath  =  os.path.join('.', 'web', 'images')
    Axis      =  sys.argv[3]
    startTime =  sys.argv[4]
    endTime   =  sys.argv[5]
    receta    =  sys.argv[6]
    datatype  =  sys.argv[7] if len(sys.argv) > 7 else "fan"
    vib_data  =  np.array([])
    tmpList   =  []
    tmpDict   =  {}
    startTime_datetime = datetime.datetime.strptime(startTime, '%Y-%m-%d_%H%M%S')
    tmpType = "" if datatype == "fan" else datatype
    saveFrequencyPath = os.path.join('.', 'data', sys.argv[2] + '/' + sys.argv[2] + '-' + sys.argv[3] + tmpType +'-SFFT.csv') #./data/OA20180910-101-F/OA20180910-101-F-X-SFFT.csv
    saveSignalPath   = os.path.join('.', 'data',  sys.argv[2] + '/' + sys.argv[2] + '-' + sys.argv[3] + tmpType + '-Time.csv') #./data/OA20180910-101-F/OA20180910-101-F-X-Time.csv

    if sys.argv[2][:2] == 'OA':
        axis = '500401'
    elif sys.argv[2][:2] == 'OB':
        axis = '500402'
        if datatype == 'vacuum':
            axis = '500404'
        elif datatype == 'water':
            axis = '500405'
    elif sys.argv[2][:2] == 'OC':
        axis = '500403'
    
    fileGroup = findVibrationFile(sys.argv[1], startTime, endTime, axis)
    
    for filename in fileGroup:
        df = pd.read_csv(filename,names=['id','timestamp','X','Y','Z','index'])
        vib_data = np.append(vib_data,df[Axis].values*1000)
    
    plt.rcParams.update({'font.size': 20})
    plt.rcParams['figure.figsize'] = (38, 18)
    f, t, Sxx = signal.spectrogram(vib_data, 1000,scaling = 'spectrum' , window = signal.get_window(('tukey',0.125),512),noverlap = 256)
    
    vibrationTime = pd.DataFrame(t, columns=['timespan'])
    for index, row in vibrationTime.iterrows():
        tmpList.append(startTime_datetime + datetime.timedelta(0,0,row['timespan']*1000000))
    vibrationTime[receta] = tmpList

    tmpList = []
    
    # plot figure
    fig = plt.figure()
    plt.pcolormesh(vibrationTime[receta], f, Sxx,vmax = 110)
    plt.ylabel('Frequency [Hz]')
    plt.xlabel('Time [sec]')
    plt.colorbar()
    # check diretory exist
    if not os.path.isdir(savePath):
        os.mkdir(savePath)
    savePath = os.path.join(savePath, sys.argv[2] + '-SFFT-' + sys.argv[3] + ('-'+datatype if datatype != "fan" else "") +'.png')
    plt.savefig(savePath,bbox_inches='tight')
    
    # save SFFT signal to csv file
    frequency_selected = select_frequecy(receta,Axis,datatype)
    mean,std     =  computeZscore(receta,Axis,datatype)

    for frequency in frequency_selected:
        tmpDict[frequency]  = Sxx[int(frequency),:]
    
    feq_signal = pd.DataFrame(tmpDict)
    tmpDict = {}

    vibrationTime[str(mean)] = 0
    vibrationTime[str(std)]  = 0
    feq_signal.to_csv(saveFrequencyPath, sep=',')
    vibrationTime.to_csv(saveSignalPath, sep=',')
    print("generate SFFT figure successful!\n")
if __name__ == "__main__":
    main()