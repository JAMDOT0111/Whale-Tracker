import pandas as pd
import xgboost as xgb
import numpy as np
import sys
import json
import argparse

def main():
    parser = argparse.ArgumentParser()
    parser.add_argument('--csv', required=True, help='Path to target.csv')
    parser.add_argument('--model', required=True, help='Path to eth_fraud_detection_model.json')
    args = parser.parse_args()

    try:
        df = pd.read_csv(args.csv)
    except FileNotFoundError:
        print(json.dumps({"error": "target.csv not found"}))
        sys.exit(1)

    target_address = df['Address'].iloc[0]
    
    cols_to_drop = [
        'Address', 'Flag', 
        'ERC20 total Ether sent contract', 'ERC20 uniq sent addr.1', 
        'ERC20 uniq rec contract addr', 'ERC20 avg time between rec 2 tnx', 
        'ERC20 avg time between contract tnx', 'ERC20 min val sent contract', 
        'ERC20 max val sent contract', 'ERC20 avg val sent contract'
    ]
    # Strip spaces first, as CSV might have ' ERC20...'
    df.rename(columns=lambda x: x.strip(), inplace=True)
    df.rename(columns={'total transactions (including tnx to create contract)': 'total transactions'}, inplace=True)
    
    # Strip from cols_to_drop as well, since we just stripped df.columns
    cols_to_drop = [c.strip() for c in cols_to_drop]
    df = df.drop(columns=cols_to_drop, errors='ignore')

    num_cols = df.select_dtypes(include=[np.number]).columns
    df[num_cols] = df[num_cols].replace(['inf', '-inf', 'Infinity'], np.nan).astype(np.float32)
    df[num_cols] = df[num_cols].replace([np.inf, -np.inf], np.nan)

    cat_cols = ['ERC20 most sent token type', 'ERC20_most_rec_token_type']
    for col in cat_cols:
        if col in df.columns:
            df[col] = df[col].fillna("None").astype(str).astype('category')

    try:
        model = xgb.XGBClassifier()
        model.load_model(args.model)
    except Exception as e:
        print(json.dumps({"error": f"Model load failed: {str(e)}"}))
        sys.exit(1)

    expected_features = model.get_booster().feature_names
    for f in expected_features:
        if f not in df.columns:
            df[f] = 0
            
    # Also ensure the categorical type is assigned to any newly added cat cols
    for col in cat_cols:
        if df[col].dtype != 'category':
             df[col] = df[col].astype(str).astype('category')

    df = df[expected_features]

    prediction = int(model.predict(df)[0])
    prob_normal = float(model.predict_proba(df)[0][0])
    prob_phishing = float(model.predict_proba(df)[0][1])

    result = {
        "address": target_address,
        "is_fraud": prediction == 1,
        "confidence": prob_phishing if prediction == 1 else prob_normal,
        "prob_phishing": prob_phishing,
        "prob_normal": prob_normal
    }
    print(json.dumps(result))

if __name__ == "__main__":
    main()
