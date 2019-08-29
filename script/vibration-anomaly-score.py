import numpy as np
import pandas as pd
import sys
import os
from sklearn.externals import joblib
from sklearn.decomposition import PCA
def checkModelPath(modelList):
    for modelPath in modelList:
        if not os.path.exists(modelPath):
            return False
    return True
def main():
    OvenName    = sys.argv[1][0:2]
    Axis        = sys.argv[2]
    datatype    = sys.argv[4]
    tmptype = "" if datatype == "fan" else datatype
    SignalPath  = os.path.join('.', 'data', sys.argv[1] + '/' + sys.argv[1] + '-' + sys.argv[2] + tmptype + '-SFFT.csv')#./data/OA20180910-101-F/OA20180910-101-F-X-SFFT.csv
    TimePath    = os.path.join('.', 'data', sys.argv[1] + '/' + sys.argv[1] + '-' + sys.argv[2] + tmptype + '-Time.csv')#./data/OA20180910-101-F/OA20180910-101-F-X-Time.csv
    try:
        SignalDataFrame         = pd.read_csv(SignalPath)
        SignalDataFrame         = SignalDataFrame.drop(SignalDataFrame.columns[0],axis=1)
        TimeDataFrame           = pd.read_csv(TimePath)

        baseName =  [TimeDataFrame.columns.values[2], OvenName, Axis]
        if datatype == "water" or datatype == "vacuum":
            baseName.insert(2, datatype)
        baseName = '-'.join(baseName)
        isolation_modelPath     = os.path.join('.', 'model', baseName + '-ios.pkl')
        first_scalarPath        = os.path.join('.', 'model', baseName + '-fst.pkl')
        second_scalarPath       = os.path.join('.', 'model', baseName + '-sec.pkl')
        
        if not checkModelPath([isolation_modelPath,first_scalarPath,second_scalarPath]):
            isolation_modelPath = os.path.join('.','default', 'model', baseName + '-ios.pkl')
            first_scalarPath    = os.path.join('.','default', 'model', baseName + '-fst.pkl')
            second_scalarPath   = os.path.join('.','default', 'model', baseName + '-sec.pkl')
        isolation_model         = joblib.load(isolation_modelPath)
        first_scalar            = joblib.load(first_scalarPath)
        second_scalar           = joblib.load(second_scalarPath)
    except Exception as e:
        print(e)
        sys.exit(1)

    first_np_scaled = first_scalar.transform(SignalDataFrame)
    data = pd.DataFrame(first_np_scaled)
    
    pca = PCA(n_components=2)
    data = pca.fit_transform(data)

    second_np_scaled = second_scalar.transform(data)
    data = pd.DataFrame(second_np_scaled)
    
    TimeDataFrame['anomaly'] = isolation_model.predict(data)
    TimeDataFrame['anomaly'] = TimeDataFrame['anomaly'].map( {1: 0, -1: 1} )
    anomaly_number = TimeDataFrame['anomaly'].value_counts()
    TimeDataFrame['anomalyscore'] = isolation_model.score_samples(data)
    
    try:
        anomaly_rate = anomaly_number[1] / anomaly_number[0]
        mean         = float(TimeDataFrame.columns.values[3])
        std          = float(TimeDataFrame.columns.values[4])
        anomaly_score   = (anomaly_rate - mean) / std
    except:
        anomaly_score = 0
    

    print(anomaly_score)
if __name__ == "__main__":
    main()