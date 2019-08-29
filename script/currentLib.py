import sys
import numpy as np
import pandas as pd
import os
import time
from sklearn.externals import joblib
import json
from sklearn.gaussian_process import GaussianProcessRegressor
from sklearn.gaussian_process.kernels import DotProduct, WhiteKernel, RBF, ConstantKernel
import random
import datetime
from sklearn.metrics import mean_squared_error, r2_score

def getCurrentData(Dir, fileName='current_fan.csv'):
    filepath = os.path.join(Dir, fileName)
    try:
        if not os.path.isfile(filepath):
            raise Exception(filepath + ' is not exist!')
        df = pd.read_csv(
            filepath, 
            encoding='utf-8',
            dtype= {
                'addr':'int64',
                'value':'float64'
            },
            parse_dates= ['timestamp']
        )
    except Exception as e:
        #print(e)
        return pd.DataFrame()
    
    addrs = df.addr.unique()
    if (not np.all([ad in addrs for ad in [0,10,20]])):
        #print(Dir, ' current doesn't have 0,10,20 addrs')
        return pd.DataFrame()
    
    if np.count_nonzero([df[df.addr == ad].value.mean() == 0.0 for ad in [0,10,20]]) > 0:
        #print(Dir, ' current value all zero !')
        return pd.DataFrame()
    
    return df

def getCuringData(Dir, CuringName):
    filepath = os.path.join(Dir, 'Calc_'+CuringName+'.csv')
    try:
        if not os.path.isfile(filepath):
            raise Exception(filepath + ' is not exist!')
        receta = pd.read_csv(filepath, encoding='ISO-8859-1', nrows=0).columns[1]
        df = pd.read_csv(filepath, encoding='ISO-8859–1', skiprows=[0,1,2,4])
        df = df.drop(columns=df.columns[-1], axis=1)
        if df.shape[0] == 0:
            raise Exception(filepath + ' is nodata!')
        df.recipe = receta
    except Exception as e :
        print(e)
        return pd.DataFrame()
    df.curingName = CuringName
    return df

def curing_preprocessing(curing, columns=['AMV', 'PMV']):
    slope = {}
    for col in columns:
        _t = np.convolve(curing[col].values, np.array([1,-1]), mode="vaild")
        slope[col+"_slope"] = np.insert(_t, 0, _t[0], axis=0)
    __columns = columns + list(slope.keys())
    
    if not 'timestamp' in __columns:
        __columns = ['timestamp'] + __columns
    
    curing_n = curing.assign(timestamp=pd.to_datetime(curing.Fecha + ' ' + curing.Hora, format="%d/%m/%Y %H:%M:%S"), **slope)[ __columns ]
    curing_n.recipe = curing.recipe
    curing_n.curingName = curing.curingName
    
    return curing_n

def current_preprocessing(current, drop_duplicated=False, drop_zero=True, mean_size=0, mean_method='mean', drop_do_average=True):
    current = current.copy()
    if drop_zero:
        current = current[current.value != 0]   
    # mark duplicated
    dup_count = 0
    cur = current
    current_n = []
    while True:
        __tmp = cur[~cur.duplicated(keep='first', subset=['timestamp', 'addr'])]
        current_n += [__tmp.assign(dup=dup_count)]
        dup_count += 1
        cur = cur[cur.duplicated(keep='first', subset=['timestamp', 'addr'])]
        
        if cur.empty:
            break
    
    extra_columns = []
    if drop_duplicated:
        if drop_do_average:    
            __tmp = current_n[0]
            for i in range(1, len(current_n)):
                __tmp = __tmp.merge(current_n[i], how='left', on=['timestamp', 'addr'], suffixes=('', str(i)))
            current_n = __tmp
            
            all_values = current_n[current_n.columns.to_series().filter(like='value')].values
            current_n = current_n.assign(
                value=np.nanmean(all_values, axis=1),
                var = np.nanvar(all_values, axis=1)
            )
            extra_columns = ['var']
        else:
            current_n = current_n[0]
    else:
        current_n = pd.concat(current_n)
        current_n = current_n.sort_values(by=['timestamp', 'dup', 'addr']) 
    
    # tranpose addr
    addrs = current_n.addr.unique()
    #addrs = np.delete(addrs, np.where(addrs == 35)[0])
    
    if addrs.shape[0] == 0:
        return pd.DataFrame()
    __tmp = [current_n[current_n.addr == addr][
        ['timestamp', 'dup', 'value'] + extra_columns
    ].rename(
        columns={s:s+'_'+str(addr) for s in ['value']+extra_columns}
    ) for addr in addrs]
    
    current_n = __tmp[0]
    for i in range(1,len(__tmp)):
        current_n = current_n.merge(__tmp[i], how='left', on=['timestamp', 'dup'])
    current_n = current_n.fillna(method='ffill')
    
    if mean_size > 0 and mean_size < current_n.shape[0]:
        cols = ['value'] + extra_columns
        cur = current_n[[s+'_'+str(addr) for s in cols for addr in addrs]].values
        
        if mean_method and mean_method.lower() == "rms":
            cur = cur ** 2
        
        new_values = np.concatenate([np.convolve(cur[:,i], np.ones(mean_size)/mean_size,mode='vaild')[:,None] for i in range(cur.shape[1])], axis=1)
        
        frontend = int((mean_size - 1) / 2)
        backend = mean_size - 1 - frontend
        
        cur[frontend:-backend, :] = new_values
        new_values = cur
        
        if mean_method and mean_method.lower() == "rms":
            new_values = np.sqrt(new_values)
        
        current_n = current_n.assign(**{s+'_'+str(addr):new_values[:, idx + i*len(cols)] for i, s in enumerate(cols) for idx, addr in enumerate(addrs)})
    
    return current_n

def mergeCuringCurrent(curing, current, extend=False):
    if curing.empty or current.empty:
        return pd.DataFrame()
    if extend:
        if type(extend) == type(""):
            how = extend
        else:
            how = 'outer'
    else:
        how = 'inner'
    cc = pd.merge(curing, current, how=how, on='timestamp', sort=True)
    if extend:
        cc = cc.fillna(method='ffill')
        cc = cc.fillna(method='bfill')
    cc.recipe = curing.recipe
    cc.curingName = curing.curingName
    
    cc = cc.dropna(axis=0)
    
    return cc

import re
def ToolDataset(dataDir, device, chooseAutoclave=None, chooseRecipe=None, startDate=None, endDate=None):
    #
    '''
    data
    - curing name (i.e OA20180829-001)
    '''
    curing_datas = []
    current_fan_datas = []
    current_heater_datas = []
    
    data_indexs = []
    
    ignore_autoclave = 0
    ignore_recipe = 0
    
    for curing_name in os.listdir(dataDir):
        result = re.search('(\w{2})(\d{8})-(\d{3})', curing_name)
        if result:
            clave, date, order = result.groups()
        else:
            #raise AttributeError('Error curing Name ({})'.format(curing_name))
            continue
        
        if (startDate is not None) and startDate > date:
            continue
        
        if (endDate is not None) and endDate < date:
            continue
        
        if (chooseAutoclave is not None) and clave != chooseAutoclave:
            ignore_autoclave += 1
            continue
        
        if not os.path.isfile(os.path.join(dataDir, curing_name, 'Calc_'+clave+date+'-'+order+'.csv')):
            continue
        
        curing = getCuringData(os.path.join(dataDir, curing_name), clave+date+'-'+order)
        if curing.empty:
            continue
            
        curing = curing_preprocessing(curing)
        
        if (chooseRecipe is not None) and curing.recipe != chooseRecipe:
            ignore_recipe += 1
            continue
        
        if device == "fan":
            current_fan = getCurrentData(os.path.join(dataDir, curing_name), fileName="current_fan.csv")
            current_heater = pd.DataFrame()
            
            if current_fan.empty:
                continue
        elif "heater" in device:
            current_fan = pd.DataFrame()
            heaterid = "".join(device.split("-")[1:2])
            current_heater = getCurrentData(os.path.join(dataDir, curing_name), fileName="current_heater{}.csv".format(heaterid))
            
            if current_heater.empty:
                continue
        else:
            raise AttributeError("No device type for "+device)
        
        curing_datas.append(curing)
        current_fan_datas.append(current_fan)
        current_heater_datas.append(current_heater)
        
        data_indexs.append([clave, date, order, curing.recipe])
    
    data_indexs = pd.DataFrame(data_indexs, columns=['autoclave', 'date', 'order', 'recipe'])
    
    if data_indexs.empty:
        if ignore_autoclave and ignore_recipe:
            raise FileNotFoundError('No data for {} {}'.format(chooseAutoclave, chooseRecipe))
        elif ignore_autoclave:
            raise FileNotFoundError('No data for {}'.format(chooseAutoclave))
        elif ignore_recipe:
            raise FileNotFoundError('No data for {}'.format(chooseRecipe))
    
    return data_indexs, curing_datas, current_fan_datas, current_heater_datas
    
def AIDCDataset(curingDir, currentDir, autoclaves = ['OA', 'OB', 'OC'], startDate=None, endDate=None):
    curing_datas = []
    current_fan_datas = []
    current_heater_datas = []
    data_indexs = []
    for autoclave in autoclaves:
        dirpath = os.path.join(curingDir, autoclave)
        a = list(os.listdir(dirpath))
        a.sort()
        for filename in a:
            # check file name fitting format
            result = re.search('Calc_(\w{2})(\d{8})-(\d{3}).csv', filename)
            # no match
            if result == None:
                continue
            clave, date, order = result.groups()
            if clave != autoclave:
                continue
            
            if (endDate is not None) and date > endDate:
                continue
                
            if (startDate is not None) and date < startDate:
                continue
            
            #print(clave, date, order)
            
            # get curing data
            dirpath = os.path.join(curingDir, autoclave)
            curing = getCuringData(dirpath, clave+date+'-'+order)
            if curing.empty:
                continue
                
            # preprocess data
            curing = curing_preprocessing(curing)
            
            # get crrent fan and heater data
            dirpath = os.path.join(currentDir, autoclave, clave+date+'-'+order)
            current_fan = getCurrentData(dirpath, fileName="current_fan.csv")
            current_heater = getCurrentData(dirpath, fileName='current_heater.csv')
            
            if (current_fan.empty or current_heater.empty):
                continue
                
            curing_datas.append(curing)
            current_heater_datas.append(current_heater)
            current_fan_datas.append(current_fan)
            
            data_indexs.append([clave, date, order, curing.recipe])        
            #print(data_indexs)
    data_indexs = pd.DataFrame(data_indexs, columns=['autoclave', 'date', 'order', 'recipe'])
    #data_indexs.curing_datas = curing_datas
    #data_indexs.current_heater_datas = current_heater_datas
    #data_indexs.current_fan_datas = current_fan_datas
    
    return data_indexs, curing_datas, current_fan_datas, current_heater_datas
        
__fanModelFileName = "fanCurrentModel.joblib"
__heaterModelFileName = "heaterCurrentModel.joblib"

def saveModel(model, mDir, device):
    fn = None
    if device == 'fan':
        fn = __fanModelFileName
    elif "heater" in device:
        fn = __heaterModelFileName
    else:
        raise AttributeError("No device type for "+device)
    
    filepath = os.path.join(mDir, fn)
    joblib.dump(model, filepath)
    return filepath

def loadModel(mDir, device):
    fn = None
    if device == 'fan':
        fn = __fanModelFileName
    elif "heater" in device:
        fn = __heaterModelFileName
    else:
        raise AttributeError("No device type for "+device)
    
    fp = os.path.join(mDir, fn)
    if os.path.isfile(fp):
        return joblib.load(fp)
    else:
        return {}

def getDataFromRaw(cc, x_feature=None, y_feature=None):
    if not x_feature:
        x_feature = ['PMV', 'AMV']
    if not y_feature:
        y_feature = ['value_'+str(i*10) for i in range(3)]
    return np.array(cc[x_feature]), np.array(cc[y_feature])

import math
def divideData(di, size=None, method='random'):
    """
    size is mean how many data as train data
    - None: half of data
    method is mean how to choose train data
    - early: from early date choose
    - random: random choose
    - later: from later date choose
    """
    di = di.sort_values(by=['date', 'recipe', 'order', 'autoclave'])
    
    if size is None:
        size=math.ceil(len(di)/2)
    
    choice = np.array([False]*len(di))
    
    if method == 'random':
        choice[np.random.choice(
            len(di), size=size, replace=False
        )] = True
    elif method == 'early':
        choice[:size] = True
    elif method == 'later':
        choice[-size:] = True

    train_di = di.iloc[choice]
    test_di = di.iloc[~choice]
    
    # if no test_data, just use train_data
    if test_di.empty:
        test_di = train_di
    
    return train_di, test_di

__kernel = WhiteKernel(10) + DotProduct(100) + RBF(10)
__max_train_size = 5000
def makeModel(cc_datas, recipe_list, x_feature=None, y_feature=None, show=False, down_sampling=True, n_restarts_optimizer=0):
    model = {}
    all_x = []
    all_y = []
    copy_x = False
    
    
    # make model each recipe
    for item in recipe_list:
        indexs = item['indexs']
        train_i = item['train_i']
        show_msg = [ item['name'] ]
        
        train_x, train_y = getDataFromRaw(cc_datas[indexs[train_i]], x_feature, y_feature)
        
        if down_sampling and train_x.shape[0] > __max_train_size:
            show_msg += ['origin: train x : {}, train y : {}'.format(train_x.shape, train_y.shape)]
            rand_idxs = np.sort(np.random.randint(0, train_x.shape[0], __max_train_size))
            train_x = train_x[rand_idxs, :]
            train_y = train_y[rand_idxs, :]
            
        show_msg += ['train x : {}, train y : {}'.format(train_x.shape, train_y.shape)]
            
        
        if show:
            print('\n'.join(show_msg))
        
        gpr = GaussianProcessRegressor(
            kernel=__kernel, copy_X_train=copy_x,
            n_restarts_optimizer = n_restarts_optimizer
        )
        gpr.fit(train_x, train_y)
        mean, std = gpr.predict(train_x, return_std=True)
        std = std[:,None]
        gpr.train_r2_ = r2_score(train_y, mean)
        gpr.train_mse_ = mean_squared_error(train_y, mean)
        
        model[item['name']] = gpr
        
        if show:
            print('\tgpr kernel : ', gpr.kernel_, ', r2 : ', gpr.train_r2_,
                 ' mse : ', gpr.train_mse_)
        
        all_x += [train_x]
        all_y += [train_y]
    
    # make model for all recipe
    all_x = np.concatenate(all_x, axis=0)
    all_y = np.concatenate(all_y, axis=0)
    
    
    
    # if data set to big, don't train kernel parameter
    optimizer = {}
    all_kernel = __kernel
    if all_x.shape[0] >= __max_train_size*2:
        optimizer['optimizer'] = None
        # get each model parameter
        all_theta = [gpr.kernel_.theta for _, gpr in model.items()]
        
        all_theta = np.array(all_theta)
        all_theta = np.mean(all_theta, axis=0)
        all_kernel = __kernel.clone_with_theta(all_theta)
        
        rand_idxs = np.sort(np.random.randint(0, all_x.shape[0], __max_train_size*2))
        
        all_x = all_x[rand_idxs, :]
        all_y = all_y[rand_idxs, :]
    
    if show:
        print('ALL RECIPE')
        print('train x : ', all_x.shape, ', train y : ', all_y.shape)    
        
    gpr = GaussianProcessRegressor(
        kernel=all_kernel, copy_X_train=copy_x,
        n_restarts_optimizer = n_restarts_optimizer, **optimizer)
    gpr.fit(all_x, all_y)
    mean, std = gpr.predict(all_x, return_std=True)
    std = std[:,None]
    gpr.train_r2_ = r2_score(all_y, mean)
    gpr.train_mse_ = mean_squared_error(all_y, mean)
    
    model['all'] = gpr
    
    if show:
        print('\tgpr kernel : ', gpr.kernel_, ', r2 : ', gpr.train_r2_,
             ' mse : ', gpr.train_mse_)
    
    return model