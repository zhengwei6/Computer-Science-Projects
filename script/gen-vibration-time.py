import sys
import glob
import pandas as pd
import time
import os
import datetime
import json

def curing_preprocessing(curing, columns):
    curing_timestamp = [time.strftime( "%Y-%m-%d", time.strptime(fecha, "%d/%m/%Y"))+' '+hora for fecha,hora in zip(curing.Fecha, curing.Hora)]
    
    if not 'timestamp' in columns:
        columns = ['timestamp'] + columns
    
    curing_n = curing.assign(timestamp=curing_timestamp)[ columns ]
    
    return curing_n

def main():
    filepath = os.path.join('.', sys.argv[1], "Calc_" + sys.argv[1][5:19] + '.csv')

    try:
        receta = pd.read_csv(filepath, encoding='ISO-8859-1', nrows=0).columns[1]
        cur_pd = pd.read_csv(filepath, encoding='ISO-8859–1', skiprows=[0,1,2,4])
        cur_pd = cur_pd.drop(columns=cur_pd.columns[-1], axis=1)

        if cur_pd.shape[0] == 0:
            print('\n load ' + filepath + ' is nodata!')
        cur_pd.recipe = receta
    except Exception as e :
        print(e)
        sys.exit(1)

    # Check recipe model exist
    '''
    modelPath = os.path.join('.', 'model', cur_pd.recipe + '-*-*.pkl')
    if len(glob.glob(modelPath)) == 0:
        print('No model for ' + cur_pd.recipe)
        sys.exit(1)
    '''
    cur_pd = curing_preprocessing(cur_pd, ['PMV', 'AMV'])
    cur_pd['timestamp'] = pd.to_datetime(cur_pd['timestamp'])
    
    #start_time = cur_pd.loc[0,'timestamp'] + datetime.timedelta(hours=1)
    start_time = cur_pd.loc[0,'timestamp']
    start_time = start_time.strftime("%Y-%m-%d_%H%M%S")
    #end_time = cur_pd.loc[cur_pd.index[-1],'timestamp'] - datetime.timedelta(hours=1)
    end_time   = cur_pd.loc[cur_pd.index[-1],'timestamp']
    end_time = end_time.strftime("%Y-%m-%d_%H%M%S")

    timeData = {'receta' : receta, 'start_time' : start_time, 'end_time' : end_time }
    timeDataJSON = json.dumps(timeData, ensure_ascii=False)
    print(timeDataJSON)
if __name__ == "__main__":
    main()

