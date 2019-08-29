# load data & model and export predict result to csv file
import sys
import numpy as np
import pandas as pd
import os
import json
import re
from currentLib import *

if __name__ == "__main__":
    curingName, dataFileName, sDir, tDir, mDir = sys.argv[1:]
    print(curingName, dataFileName, sDir, tDir, mDir)
    
    # check curing name
    result = re.search('(\w{2})(\d{8})-(\d{3})', curingName)

    if result:
        curingName = result.group()
    else:
        print('Error curing Name ({})'.format(curingName))
        sys.exit(-1)
    
    autoclave = curingName[:2]
    filepath = os.path.join(tDir, dataFileName+'.csv')
    jsonfilepath = os.path.join(tDir, dataFileName+'.json')

    # fetch Data
    curing = getCuringData(sDir, curingName)
    if (curing.empty):
        sys.exit(-1)
    # use filename to decide what heater data we need
    heaterid = "".join(dataFileName.split("-")[2:3])
    current = getCurrentData(sDir, "current_heater{}.csv".format(heaterid))
    if (current.empty):
        sys.exit(-1)
    
    # preprocess Data
    curing = curing_preprocessing(curing)
    current = current_preprocessing(
        current, drop_zero=False, mean_size=60, 
        mean_method='rms', drop_duplicated=True
    )
    # represent Data
    columns_x = ["PMV", "AMV", "AMV_slope"] 
    columns_y = ['value_'+str(addr) for addr in [0,10,20]]

    # combine Data
    cc = mergeCuringCurrent(curing, current)
    if (cc.empty):
        print('Combine curing and current data error!')
        sys.exit(-1)

    x = np.array(cc[columns_x])
    y = np.array(cc[columns_y])
    
    # store some info about model
    model_info = {}

    # find model
    def findModel(mDir, autoclave, recipe, hid):
        models = loadModel(mDir, 'heater')
        model = None
        if autoclave in models:
            #model = models[autoclave]

            if recipe + hid in models[autoclave]:
                model = models[autoclave][recipe+hid]
                model_info['model_autoclave'] = autoclave
                model_info['model_recipe'] = recipe
            elif 'all' + hid in models[autoclave]:
                model = models[autoclave]['all'+hid]
                model_info['model_autoclave'] = autoclave
                model_info['model_recipe'] = 'all'
        return model

    model = findModel(mDir, autoclave, curing.recipe, heaterid)

    # find model in default
    if model is None:
        model = findModel(os.path.join('./default', mDir), autoclave, curing.recipe, heaterid)

    # check result exist
    if os.path.isfile(filepath) and os.path.isfile(jsonfilepath):
        with open(jsonfilepath, 'r') as f:
            prev_model_info = json.load(f)
            print('prev model info :', prev_model_info, 'current model info :', model_info)
            if 'model_recipe' in model_info and 'model_recipe' in prev_model_info and model_info['model_recipe'] == prev_model_info['model_recipe']:
                print('Already finish!')
                sys.exit(0)
            else:
                print('Detect new model, do analysis')

    # predict data
    columns_model = []
    if model is not None:
        mean, std = model.predict(x, return_std=True)
        std = std[:,None]

        # insert to cc dataframe
        cc = cc.assign(
            mean_0=mean[:,0],
            mean_10=mean[:,1],
            mean_20=mean[:,2],
            std=std[:,0],
        )

        # ignore result for zero current
        from functools import reduce
        value_mask = reduce(lambda x,y: x & y, [cc[y] == 0 for y in columns_y])
        cc.loc[value_mask, ['mean_0', 'mean_10', 'mean_20', 'std']] = np.nan

        columns_model = ['mean_'+str(addr) for addr in [0,10,20]] + ['std']
        for s in ['z_thr']:
            model_info['model_' + s] = list(getattr(model,s))

        y = np.array(cc.loc[~value_mask, columns_y])
        mean = np.array(cc.loc[~value_mask, ['mean_0', 'mean_10', 'mean_20']])
        std = np.array(cc.loc[~value_mask, ['std']])
        z = (np.abs(y - mean)/std)
        model_info['z_mean'] = list(np.mean(z, axis=0))
        model_info['z_std'] = list(np.std(z, axis=0))
        model_info['z_score'] = list(np.percentile(z, 95, axis=0))
    else:
        print('No model can use !')
    
    # check out dir
    if not os.path.isdir(tDir):
        os.mkdir(tDir)

    cc[['timestamp']+columns_x+columns_y+columns_model].to_csv(filepath, index=False)
    with open(jsonfilepath, 'w') as f:
        json.dump(model_info,f)
    print('analysis doen, csv file at ', filepath, ' and json file at', jsonfilepath)
    