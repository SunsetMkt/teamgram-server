/*
 * WARNING! All changes made in this file will be lost!
 *   Created from by 'dalgen'
 *
 * Copyright (c) 2021-present,  Teamgram Studio (https://teamgram.io).
 *  All rights reserved.
 *
 * Author: teamgramio (teamgram.io@gmail.com)
 */

package dataobject

type UserQtsUpdatesDO struct {
	Id         int64  `db:"id"`
	UserId     int64  `db:"user_id"`
	Qts        int32  `db:"qts"`
	UpdateType int32  `db:"update_type"`
	UpdateData []byte `db:"update_data"`
	Date2      int64  `db:"date2"`
}