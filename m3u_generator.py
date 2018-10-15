#!/usr/bin/python3
# -*- coding: UTF-8 -*-


from codecs import open
from contextlib import closing
from datetime import datetime
from datetime import timedelta
from os import makedirs
from os.path import dirname
from socket import gethostname, gethostbyname
from sys import stderr
from time import sleep
from traceback import print_exc, format_exc

from channel_handler import ChannelHandler
from config import Config
from utils import Utils


class M3UGenerator:

    @staticmethod
    def main():
        while True:
            print('Started at', datetime.now().strftime('%b %d %H:%M:%S'), end='\n\n')

            Utils.wait_for_internet()

            data_set_number = 0

            for data_set in Config.DATA_SETS:
                data_set_number += 1
                print('Processing data set', data_set_number, 'of', len(Config.DATA_SETS))

                out_file_name = data_set.get('OUT_FILE_NAME')
                out_file_encoding = data_set.get('OUT_FILE_ENCODING')
                out_file_first_line = data_set.get('OUT_FILE_FIRST_LINE')

                makedirs(dirname(out_file_name), exist_ok=True)

                with closing(open(out_file_name, 'w', out_file_encoding)) as out_file:
                    out_file.write(out_file_first_line)

                    total_channel_count = 0
                    allowed_channel_count = 0

                    channel_list = ChannelHandler.get_channel_list(data_set)
                    channel_list = ChannelHandler.replace_categories(channel_list, data_set)
                    channel_list.sort(key=lambda x: x.get('name'))
                    channel_list.sort(key=lambda x: x.get('cat'))

                    if data_set.get('CLEAN_FILTER'):
                        ChannelHandler.clean_filter(channel_list, data_set)

                    for channel in channel_list:
                        total_channel_count += 1

                        if ChannelHandler.is_channel_allowed(channel, data_set):
                            ChannelHandler.write_entry(channel, data_set, out_file)
                            allowed_channel_count += 1

                print('Playlist', data_set.get('OUT_FILE_NAME'), 'successfully generated.')
                print('Channels processed in total:', total_channel_count)
                print('Channels allowed:', allowed_channel_count)
                print('Channels denied:', total_channel_count - allowed_channel_count)

                if data_set_number < len(Config.DATA_SETS):
                    print('Sleeping for', timedelta(seconds=Config.DATA_SET_DELAY),
                          'before processing next data set...')
                    sleep(Config.DATA_SET_DELAY)

                print('')

            print('Finished at', datetime.now().strftime('%b %d %H:%M:%S'))
            print('Sleeping for', timedelta(seconds=Config.UPDATE_DELAY), 'before the new update...')
            print('-' * 45, end='\n\n\n')
            sleep(Config.UPDATE_DELAY)


# Main start point.
if __name__ == '__main__':
    # noinspection PyBroadException
    try:
        M3UGenerator.main()
    except Exception:
        print_exc()

        if Config.MAIL_ON_CRASH:
            print('Sending notification.', file=stderr)
            Utils.send_email('M3UGenerator has crashed on ' + gethostname() + '@' + gethostbyname(gethostname()),
                             format_exc())

        if Config.PAUSE_ON_CRASH:
            input('Press <Enter> to exit...\n')

        exit(1)
