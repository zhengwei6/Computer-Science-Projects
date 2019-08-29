from currentLib import *
import sys
import os
import re
import random
import pandas as pd
import numpy as np
import argparse

def preprocessing(di, t):
    cc_datas = {}
    xy_datas = {}
    for idx in di.index:
        curing = curing_datas[idx]
        
        if t.lower() == 'fan':
            current = current_fan_datas[idx]
            current = current_preprocessing(
                current, drop_duplicated=True
            )
            my_x_feature = ["PMV", "AMV"]
        elif 'heater' in t.lower() :
            current = current_heater_datas[idx]
            current = current_preprocessing(
                current, drop_zero=False, mean_size=60, mean_method='rms', drop_duplicated=True
            )
            my_x_feature = ["PMV", "AMV", "AMV_slope"]
        else:
            raise AttributeError('No for {} type'.format(t))
            
        my_y_feature = ["value_0", "value_10", "value_20"]
    
        cc = mergeCuringCurrent(curing, current, extend=False)
        if (cc.empty):
            raise AttributeError('Ouah! something wrong')
        
        if 'heater' in t.lower():
            cc = cc[cc.value_0 != 0]
            cc = cc[cc.value_10 != 0]
            cc = cc[cc.value_20 != 0]
        
        cc_datas[idx] = cc
        
        xy_datas[idx] = getDataFromRaw(cc, x_feature=my_x_feature, y_feature=my_y_feature)
    return cc_datas, xy_datas

def z_score(y_hat, mean, std):
    return  np.abs(mean - y_hat)/std

def wrapZ(y_true, mean, std):
    if (np.count_nonzero(std == 0)):
        tmpmin = 1e-5
        if len(std[std != 0]) > 0:
            if tmpmin > np.min(std[std != 0]):
                tmpmin = np.min(std[std != 0])
            
        std[std == 0] = tmpmin
    return np.mean(z_score(y_true, mean, std), axis=None)

def getModel(x, y, t):
    common_option = {
        'copy_X_train':True,
        'normalize_y':True
    }
    model_options = [
        {
            'kernel':WhiteKernel(10) + DotProduct(100) + RBF(10),
            'n_restarts_optimizer':0
        }
    ]
    
    if t.lower() == 'fan':
        model_options += [
            {
                'kernel':WhiteKernel() + DotProduct() + RBF(),
                'n_restarts_optimizer':5
            }
        ]
    elif 'heater' in t.lower():
        model_options += [
            {
                'kernel':WhiteKernel() + DotProduct() + ConstantKernel() * RBF(),
                'n_restarts_optimizer':5
            }
        ]
    else:
        raise AttributeError('No for {} type'.format(t))
    
    __models = []
    
    for o in model_options:
        __models.append(GaussianProcessRegressor(**{**o, **common_option}))
    
    __scores = []
    for m in __models:
        m.fit(x,y)
        __scores.append(m.score(x, y))
    
    model = __models[np.argmax(__scores)]
    
    return model

def __getPredict(x, y, model):
    mean, std = model.predict(x, return_std=True)
    std = std[:,None]
    
    return mean, std

def getPredict(di, models, one_for_all=False):
    predicts = []
    for idx in di.index:
        row = di.loc[idx]
        
        if one_for_all:
            m = models[row.autoclave]['all']
        else:
            m = models[row.autoclave][row.recipe]
        x, y = xy_datas[idx]
        
        predicts.append(__getPredict(x, y, m))
    
    return predicts

def algorithm1(di, one_for_all=False):
    models = {}
    for autoclave in di.autoclave.unique():
        models[autoclave] = {}
        if one_for_all:
            all_x = []
            all_y = []
        for recipe in di[di.autoclave == autoclave].recipe.unique():
            current_di = di[
                (di.autoclave == autoclave) & (di.recipe == recipe)
            ]
            x = []
            y = []
            for idx in current_di.index:
                _x, _y = xy_datas[idx]
                x.append(_x)
                y.append(_y)
            
            if one_for_all:
                all_x += x
                all_y += y
                continue
            x = np.concatenate(x, axis=0)
            y = np.concatenate(y, axis=0)
            print('Training', autoclave, recipe, ' : x size', x.shape, 'y size', y.shape)
            models[autoclave][recipe] = getModel(x, y, data_type)
        if one_for_all:
            all_x = np.concatenate(all_x, axis=0)
            all_y = np.concatenate(all_y, axis=0)
            print('Training', autoclave, 'all recipe', ' : x size', all_x.shape, 'y size', all_y.shape)
            models[autoclave]['all'] = getModel(all_x, all_y, data_type)
        
    return models

def getThreshold(di, predicts):
    zs = []
    for i, idx in enumerate(di.index):
        row = di.loc[idx]
        
        x, y = xy_datas[idx]
        mean, std = predicts[i]
        zs.append(z_score(y, mean, std))
    
    zs = np.concatenate(zs, axis=0)
    
    # average
    #thr = np.mean(zs, axis=0)
    
    # 95%
    thr = np.percentile(zs, 95, axis=0)
    
    return thr

def algorithm2(train_di, test_di, one_for_all=False):
    models = algorithm1(train_di, one_for_all=one_for_all)
    thrs = {}
    
    for autoclave in test_di.autoclave.unique():
        thrs[autoclave] = {}
        
        if one_for_all:
            current_di = test_di[
                (test_di.autoclave == autoclave)
            ]
            predicts = getPredict(current_di, models, one_for_all=True)
            thrs[autoclave]['all'] = getThreshold(current_di, predicts)
            continue
    
        for recipe in test_di[test_di.autoclave == autoclave].recipe.unique():
            current_di = test_di[
                (test_di.autoclave == autoclave) & (test_di.recipe == recipe)
            ]
            predicts = getPredict(current_di, models)
            thrs[autoclave][recipe] = getThreshold(current_di, predicts)
    
    return models, thrs


if __name__ == "__main__":
    args = argparse.ArgumentParser()
    args.add_argument('--oven', '-ov', type=str, required=False, help='Name for oven, train for each oven by default')
    args.add_argument('--recipe', '-re', type=str, required=False, help='Name for recipe, train all recipe by default')
    args.add_argument('--data', type=str, required=False, help='Diretory for curing data, default is data', default='data')
    args.add_argument('--model', type=str, required=False, help='Diretory for model, default is model', default='model')
    args.add_argument('--start', '-s', type=str, required=False, help='Curing date range for start dete')
    args.add_argument('--end', '-e', type=str, required=False, help='Curing date range for end dete')
    args.add_argument('--heater', '-heater', type=int, required=False, help='Select which heater', default=0)
    
    args = args.parse_args()
    
    dataDir = args.data
    modelDir = args.model
    chooseAutoclave = args.oven
    chooseRecipe = args.recipe
    startDate = args.start
    endDate = args.end
    heater = {0:'', 1:'1', 2:'2'}[args.heater]
    data_type='heater' + ('-' + heater if heater != "" else '')
    
    data_indexs, curing_datas, current_fan_datas, current_heater_datas \
    = ToolDataset(dataDir, data_type, chooseAutoclave, chooseRecipe, startDate, endDate)
    
    data_mask = data_indexs.autoclave != ""
    
    if startDate is not None:
        data_mask = (data_indexs.date > startDate) & data_mask
    if endDate is not None:
        data_mask = (data_indexs.date < endDate) & data_mask
    
    # get data index we need
    data_indexs = data_indexs[data_mask]
    
    if data_indexs.empty:
        raise FileNotFoundError('No data can train model, check {} has data'.format(dataDir))
    
    # preprocessing
    cc_datas, xy_datas = preprocessing(data_indexs, t=data_type)
    
    models = loadModel(modelDir, data_type)
    
    one_for_all = False
    size = None
    method = 'random'
    if chooseRecipe is None:
        one_for_all = True
        size = 6
        method = 'early'
    
    # make model
    train_data_indexs, test_data_indexs = divideData(data_indexs, size=size, method=method)
    print('Train model for '+data_type)
    print('Training curings : ')
    print(train_data_indexs)
    mymodels, mythrs = algorithm2(train_data_indexs, test_data_indexs, one_for_all)
    
    for autoclave in mymodels.keys():
        if autoclave not in models:
            models[autoclave] = {}
        if one_for_all:
            mymodels[autoclave]['all'].z_thr = mythrs[autoclave]['all']
            models[autoclave]['all' + heater] = mymodels[autoclave]['all']
            print('Put new model {} {} into modelfile'.format(autoclave, 'all'))
            break
        for recipe in mymodels[autoclave].keys():
            mymodels[autoclave][recipe].z_thr = mythrs[autoclave][recipe]
            models[autoclave][recipe + heater] = mymodels[autoclave][recipe]
            print('Put new model {} {} into modelfile'.format(autoclave, recipe))
    
    # finally store model to model directory
    filepath = saveModel(models, modelDir, data_type)
    print('Gen model done, the file at ', filepath)