import pandas as pd
import numpy as np
import argparse
import logging
import os
import glob
import time
import datetime
from scipy import signal
from scipy.signal import argrelextrema
from sklearn.ensemble import IsolationForest
from sklearn import preprocessing
from sklearn.decomposition import PCA
from sklearn.externals import joblib

# Constant
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(threadName)-12.12s] [%(levelname)-5.5s]  %(message)s",
    handlers=[
        logging.FileHandler('my.log', 'w', 'utf-8'),
        logging.StreamHandler()
    ])

logger              = logging.getLogger()
curingPath          = 'curingData'
vibrationPath       = 'vibrationData'
anomalyRatePath     = 'anomalyRate'
modelPath           = 'model'
selectFrequencyPath = 'vibrationData'
selectFrequencyName = 'Select_Frequency.csv'
sensorLabel         = {'OA' : '500401' , 'OB' : '500402' , 'OC' : '500403' }
outliers_fraction   = 0.01


# Add parser object
'''
@param   
@return  parser(object)
'''
def processCommand():
    parser = argparse.ArgumentParser()
    parser.add_argument('--oven','-ov',type=str, required=True, help = 'Name for oven')
    parser.add_argument('--recipe', '-re', type=str, required=True, help='Name for recipe')
    parser.add_argument('--axis', '-as', type=str, required=True, help='Name for axis')
    parser.add_argument('--type', '-ty', type=str, required=False, default="fan", help='Name for type')
    return parser.parse_args()
'''
@param ovenName,axisName
@return 
'''
def checkArgument(ovenName,axisName):
    if not (ovenName == 'OA' or  ovenName == 'OB' or ovenName == 'OC'):
        logger.error('oven name argument error')
        exit(1)
    if not(axisName == 'X' or axisName == 'Y' or axisName == 'Z'):
        logger.error('axis name argument error')
        exit(1)
'''
@param [curingPath, vibrationPath ,anomalyRatePath ,modelPath ,selectFrequencyPath ,selectFrequencyName]
@return 
'''
def checkFile(fileList):
    curingPath, vibrationPath ,anomalyRatePath ,modelPath ,selectFrequencyPath ,selectFrequencyName = fileList
    if not os.path.isdir(os.path.join('.',curingPath)):
        logger.error(curingPath+' directory not exist')
        exit(1)
    if not os.path.isdir(os.path.join('.',vibrationPath)):
        logger.error(vibrationPath+' directory not exist')
        exit(1)
    if not os.path.isdir(os.path.join('.',anomalyRatePath)):
        logger.error(anomalyRatePath+' directory not exist')
        exit(1)
    if not os.path.isdir(os.path.join('.',modelPath)):
        logger.error(modelPath+' directory not exist')
        exit(1)
    if not os.path.isdir(os.path.join('.',selectFrequencyPath)):
        logger.error(selectFrequencyPath+' directory not exist')
        exit(1)
    if not os.path.exists(os.path.join('.',selectFrequencyPath,selectFrequencyName)):
        logger.warning(selectFrequencyPath + ' file not exist')
        logger.warning('file not exist')
        with  open(os.path.join('.',selectFrequencyPath,selectFrequencyName), 'wb') as csvfile:
            logger.warning('create '+os.path.join('.',selectFrequencyPath,selectFrequencyName), 'wb')
        exit(1)


'''
@param   oven name(str), recipe name(str)
@return  curring data file name satisfy param (DataFrame)
'''
def findCurringData(ovenName, recipeName):

    logger.info('Start find curring data')
    # find curring data files
    filePath =  os.path.join('.', curingPath , 'Calc_' + ovenName + '*')
    try:
        files = glob.glob(filePath)
    except Exception as e:
        logger.error('Wrong curingPath')
        exit(1)
    
    # return curring data files name satisfy param 
    files.sort()
    fileDataFrame = pd.DataFrame(columns=['filename','curringname'])
    for filename in files:
        receta = pd.read_csv(filename, encoding='ISO-8859-1', nrows=0).columns[1]
        fileDataFrame = fileDataFrame.append({'filename':filename,'curringname':receta}, ignore_index=True)
    curringData = fileDataFrame.loc[(fileDataFrame['curringname'] == recipeName), ['filename']] 
    curringData = curringData.reset_index(drop=True)
    if curringData.size == 0:
        logger.error('No curring data match recipe')
        exit(1)
    return curringData

'''
@param  curring data path ,oven name
@return list of vibration data file
'''
def processCurringData(filePath, OvenName, sensorType):
    logger.info('Start process curring data: '+ filePath)

    # read curring csv to dataframe
    try:
        receta = pd.read_csv(filePath, encoding='ISO-8859-1', nrows=0).columns[1]
        cur_pd = pd.read_csv(filePath, encoding='ISO-8859–1', skiprows=[0,1,2,4])
        cur_pd = cur_pd.drop(columns=cur_pd.columns[-1], axis=1)
        if cur_pd.shape[0] == 0:
            logger.error('load ' + filePath + ' is nodata!')
            exit(1)
    except Exception as e :
        logger.error('load ' + filePath + ' fail!')
        exit(1)
    
    curing  = cur_pd
    columns = ['PMV', 'AMV']
    curing_timestamp = [time.strftime( "%Y-%m-%d", time.strptime(fecha, "%d/%m/%Y"))+' '+hora for fecha,hora in zip(curing.Fecha, curing.Hora)]
    if not 'timestamp' in columns:
        columns = ['timestamp'] + columns
    cur_pd = curing.assign(timestamp=curing_timestamp)[ columns ]
    cur_pd['timestamp'] = pd.to_datetime(cur_pd['timestamp'])
    start_time = cur_pd.loc[0,'timestamp'] + datetime.timedelta(hours=1)
    end_time = cur_pd.loc[cur_pd.index[-1],'timestamp'] - datetime.timedelta(hours=1)
    start_time_str = start_time.strftime("%Y-%m-%d_%H%M")
    end_time_str = end_time.strftime("%Y-%m-%d_%H%M")
    
    
    # find vibration data index
    vibrationFileName = start_time_str[0:10]
    sensorCode = sensorLabel[OvenName]
    if OvenName == 'OB':
        if sensorType == 'vacuum':
            sensorCode = '500404'
        elif sensorType == 'water':
            sensorCode = '500405'
    vibrationFilePath = os.path.join('.',vibrationPath,vibrationFileName,'*'+sensorCode + '.csv')
    files = []
    
    if not(start_time_str[0:10] == end_time_str[0:10]) :
        vibrationFileNameDay1 = start_time_str[0:10]
        vibrationFileNameDay2 = end_time_str[0:10] 
        
        vibrationFilePathDay1 = os.path.join('.',vibrationPath,vibrationFileNameDay1,'*'+sensorCode + '.csv')
        vibrationFilePathDay2 = os.path.join('.',vibrationPath,vibrationFileNameDay2,'*'+sensorCode + '.csv')
        try:
            filesDay1 = glob.glob(vibrationFilePathDay1)
            filesDay1.sort()
            startAppend = 0
            for i, filename in enumerate(filesDay1):
                if start_time_str in filename:
                    startAppend = 1
                if startAppend == 1:
                    files.append(filename)
                    
            filesDay2 = glob.glob(vibrationFilePathDay2)

            filesDay2.sort()
            startAppend = 1
            for i, filename in enumerate(filesDay2):
                if end_time_str in filename:
                    startAppend = 0
                if startAppend == 1:
                    files.append(filename)
        except Exception as e :
            print('load ' + vibrationFilePath + ' fail!')
            exit(1)
    else:
        vibrationFileName = start_time_str[0:10]
        vibrationFilePath = os.path.join('.',vibrationPath,vibrationFileName,'*'+sensorCode + '.csv')
        try:
            files = glob.glob(vibrationFilePath)
            files.sort()
        
            start_index = 0
            end_index = 0
            for i, filename in enumerate(files):
                if start_time_str in filename:
                    start_index = i
                elif end_time_str in filename:
                    end_index = i
        
            if len(files) == 0 or end_index <= start_index:
                print(vibrationFilePath+' error')
                return [-1]
        
        except Exception as e :
            print('load ' + vibrationFilePath + ' fail!')
            exit(1)
            
        files = files[start_index:end_index]
    if len(files)  == 0:
        return [-1]
    return files


'''
@param  list of vibration data file , training axis
@return isolation_model,first_min_max_scaler,second_min_max_scaler,frequencySelect
'''
def training_isolation_forest(vibationFileList,axis):
    logger.info('Start train isolation forest')

    vibData = np.array([])
    for filename in vibationFileList :
        df = pd.read_csv(filename,names=['id','timestamp','X','Y','Z','index'])
        vibData = np.append(vibData,df[axis].values*1000)
    f, t, Sxx = signal.spectrogram(vibData, 1000,scaling = 'spectrum' , window = signal.get_window(('tukey',0.125),512),noverlap = 0)
    
    ans = np.zeros((Sxx[:,1].size,), dtype=int)
    i = 0
    for column in Sxx.T:
        if i == 10000:
            break
        i = i + 1
        peakind = signal.find_peaks_cwt(column, np.arange(1,10))
        ans[peakind]  = ans[peakind] + 1
    frequencySelect = np.argpartition(ans , -4)[-4:]
    trainData = pd.DataFrame({'0':Sxx[frequencySelect[0],:],'1':Sxx[frequencySelect[1],:],'2':Sxx[frequencySelect[2],:],'3':Sxx[frequencySelect[3],:]})
    
    # first scaler
    first_min_max_scaler = preprocessing.StandardScaler()
    first_np_scaled = first_min_max_scaler.fit_transform(trainData)
    trainData = pd.DataFrame(first_np_scaled)
    
    # PCA
    pca = PCA(n_components=2)
    trainData = pca.fit_transform(trainData)

    # second scaler
    second_min_max_scaler = preprocessing.StandardScaler()
    second_np_scaled = second_min_max_scaler.fit_transform(trainData)
    trainData = pd.DataFrame(second_np_scaled)
    
    isolation_model =  IsolationForest(contamination = outliers_fraction,behaviour='new')
    isolation_model.fit(trainData)

    return isolation_model,first_min_max_scaler,second_min_max_scaler,frequencySelect

'''
@param  list of vibration data file , training axis
@return isolation_model,first_min_max_scaler,second_min_max_scaler,curringData,frequencySelect,axis,oven
'''
def testing_isolation_forest(isolation_model,first_min_max_scaler,second_min_max_scaler,curringData,frequencySelect,axis,oven, sensortype):
    logger.info('Testing train isolation forest')
    test_files = curringData['filename'].tolist()
    anomaly_list = []
    anomaly_name = []
    for i, val in enumerate(test_files):
        try:
            vibationFileList  = processCurringData(val,oven, sensortype)
            if vibationFileList[0] == -1:
                logger.warning(val+' data cross day or vibration file not exist')
                continue
            
            vibData = np.array([])
            for filename in vibationFileList :
                df = pd.read_csv(filename,names=['id','timestamp','X','Y','Z','index'])
                vibData = np.append(vibData,df[axis].values*1000)
            f, t, Sxx = signal.spectrogram(vibData, 1000,scaling = 'spectrum' , window = signal.get_window(('tukey',0.125),512),noverlap = 0)

            testData = pd.DataFrame({'0':Sxx[frequencySelect[0],:],'1':Sxx[frequencySelect[1],:],'2':Sxx[frequencySelect[2],:],'3':Sxx[frequencySelect[3],:]})
            test_temp = pd.DataFrame({'timestamp':t})
            # first scaler
            first_np_scaled = first_min_max_scaler.transform(testData)
            testData = pd.DataFrame(first_np_scaled)

            # PCA
            pca = PCA(n_components=2)
            testData = pca.fit_transform(testData)

            #second scaler
            second_np_scaled = second_min_max_scaler.transform(testData)
            testData = pd.DataFrame(second_np_scaled)
            
            test_temp['anomaly']      = isolation_model.predict(testData)
            test_temp['anomalyscore'] = isolation_model.score_samples(testData)
            test_temp['anomaly']      = test_temp['anomaly'].map( {1: 0, -1: 1} )

            anomaly_number = test_temp['anomaly'].value_counts()
            try:
                anomaly_score = anomaly_number[1] / anomaly_number[0]
            except Exception as e :
                anomaly_score = 0
            anomaly_name.append(val)
            anomaly_list.append(anomaly_score)
            logger.info('curring data: '+val+' anomaly rate: '+str(anomaly_score))
        except Exception as e :
            print(e)
            logger.warning(val+' test fail!')
            continue
    return anomaly_name,anomaly_list
'''
@param  anomaly filePath
@return mean , std
'''
def computeZscore(filePath):
    ZscoreDataFrame = pd.read_csv(filePath)
    return ZscoreDataFrame['anomaly_score'].mean(),ZscoreDataFrame['anomaly_score'].std(ddof=0)


def main():
    args              = processCommand()
    # check arguments
    checkArgument(args.oven,args.axis)
    # check file exists
    checkFile([curingPath, vibrationPath ,anomalyRatePath ,modelPath ,selectFrequencyPath ,selectFrequencyName])

    curringData       = findCurringData(args.oven,args.recipe)
    vibationFileList  = []
    for trainingData in curringData.loc[:,['filename']].values:
        vibationFileList  = processCurringData(trainingData[0],args.oven, args.type)
        if vibationFileList[0] == -1:
            logger.warning('Training ' + trainingData[0] + 'vibration file error and continue to find next one')
            continue
        else:
            logger.info('check vibration data of '+trainingData[0] + ' successful')
            break
    if vibationFileList[0] == -1:
        logger.error('vibration file error')
        exit(1)
    
    # train isoltion forest
    isolation_model,first_min_max_scaler,second_min_max_scaler,frequencySelect = training_isolation_forest(vibationFileList,args.axis)
    
    # testing
    anomaly_name,anomaly_list = testing_isolation_forest(isolation_model, first_min_max_scaler, second_min_max_scaler, curringData, frequencySelect, args.axis,args.oven,args.type)
    
    baseName = [args.recipe, args.oven, args.axis]
    if args.type != "fan":
        baseName.insert(2, args.type)
    baseName = '-'.join(baseName)
    # store anomaly rate

    anomalyFileName = baseName + '.csv'
    anomalyFilePath = os.path.join('.', anomalyRatePath , anomalyFileName)
    
    storeDataFrame  = pd.DataFrame({'name' : anomaly_name,'anomaly_score':anomaly_list  })
    storeDataFrame.to_csv(anomalyFilePath , sep=',')
    
    # store model
    modelFstName   = baseName + '-fst.pkl'
    modelSecName   = baseName + '-sec.pkl'
    modelIosName   = baseName + '-ios.pkl'

    modelFstPath   = os.path.join('.', modelPath , modelFstName)
    modelSecPath   = os.path.join('.', modelPath , modelSecName)
    modelIosPath   = os.path.join('.', modelPath , modelIosName)

    joblib.dump(first_min_max_scaler, modelFstPath)
    joblib.dump(second_min_max_scaler, modelSecPath)
    joblib.dump(isolation_model, modelIosPath)
    
    # store select frequency
    freqencySequence         = str(frequencySelect[0])+';'+str(frequencySelect[1])+';'+str(frequencySelect[2])+';'+str(frequencySelect[3])
    selectFrequencyFilePath  = os.path.join('.', selectFrequencyPath , selectFrequencyName)
    mean,std                 = computeZscore(anomalyFilePath)
    selectFrequencyDataFrame = pd.read_csv(selectFrequencyFilePath,names=['recipe', 'type', 'axis','frequency','mean','std'])
    tempDataFrame            = selectFrequencyDataFrame.loc[selectFrequencyDataFrame['recipe'] == args.recipe]
    tempDataFrame            = tempDataFrame.loc[tempDataFrame['axis'] == args.axis]
    tempDataFrame            = tempDataFrame.loc[tempDataFrame['type'] == args.type]
    addDataFrame             = pd.DataFrame([[args.recipe , args.type, args.axis , freqencySequence , str(mean) , str(std)]], columns=['recipe', 'type', 'axis','frequency','mean','std'])
    
    if tempDataFrame.shape[0] == 0:
        selectFrequencyDataFrame = selectFrequencyDataFrame.append(addDataFrame)
    elif tempDataFrame.shape[0] == 1:
        selectFrequencyDataFrame.at[tempDataFrame.index.values[0] , 'recipe']     = args.recipe
        selectFrequencyDataFrame.at[tempDataFrame.index.values[0] , 'type']       = args.type
        selectFrequencyDataFrame.at[tempDataFrame.index.values[0] , 'axis']       = args.axis
        selectFrequencyDataFrame.at[tempDataFrame.index.values[0] , 'frequency']  = freqencySequence 
        selectFrequencyDataFrame.at[tempDataFrame.index.values[0] , 'mean']       = str(mean)
        selectFrequencyDataFrame.at[tempDataFrame.index.values[0] , 'std']        = str(std)
    else:
        logger.warning('same value of Select_Frequency.csv')
    
    selectFrequencyDataFrame.to_csv(selectFrequencyFilePath, header=False, index=False , sep=',')
    logger.info('store Select_Frequency.csv sucessfully')

if __name__ == '__main__':
    main()
    